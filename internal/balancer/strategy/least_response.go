package strategy

import (
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"go-balancer/internal/util"
	"math"
	"net/http"
	"net/http/httptrace"
	"time"
)

const MEASUREMENT_QUEUE_SIE = 10

// Chooses backends based on recorded TTFB from previous requests.
// Tracked with an exponential moving average.
type leastResponse struct {
	// A list of queues of the last few TTFB measurements
	responseTimeMeasurements []util.Queue[time.Duration]

	// the simple moving average of TTFBs
	responseTimes []time.Duration
}

func newLeastResponse(cfg config.StrategyConfig, bm *backend.BackendManager) *leastResponse {
	queues := make([]util.Queue[time.Duration], bm.GetBackendCount())
	for i := range queues {
		queues[i] = util.NewRingBufferQueue[time.Duration](MEASUREMENT_QUEUE_SIE)
	}

	return &leastResponse{
		responseTimeMeasurements: queues,
		responseTimes:            make([]time.Duration, bm.GetBackendCount()),
	}
}

func (lr *leastResponse) applyResponseTimeUpdate(backendIndex int, responseTime time.Duration) {
	count := int64(lr.responseTimeMeasurements[backendIndex].Count())

	if count == 0 {
		// if no measurements, use measurement as average
		lr.responseTimes[backendIndex] = responseTime
		lr.responseTimeMeasurements[backendIndex].Enqueue(responseTime)
	} else if count == MEASUREMENT_QUEUE_SIE {
		// if queue full, discard oldest measurement
		old := lr.responseTimeMeasurements[backendIndex].Dequeue()

		// calculate change in average
		change := (responseTime - old) / MEASUREMENT_QUEUE_SIE

		// apply change
		lr.responseTimes[backendIndex] += change

		lr.responseTimeMeasurements[backendIndex].Enqueue(responseTime)
	} else {
		// queue not quite full, update avg with others

		// current avg will be (x_i+...+x_{count-1})/count
		// new is therefore ((curr*count)+new)/(count+1)
		lr.responseTimes[backendIndex] = time.Duration(((int64(lr.responseTimes[backendIndex]) * count) + int64(responseTime)) / (count + 1))

		lr.responseTimeMeasurements[backendIndex].Enqueue(responseTime)
	}
}

// Modify the request to attach a client trace and measure TTFB
func (lr *leastResponse) ModifyRequest(backendIndex int, r *http.Request) *http.Request {
	// declare the start time so we can capture in the trace
	// wait to initialise until after the trace context is set up to make timing more accurate
	var startTime time.Time
	defer func() { startTime = time.Now() }()

	// attach a function to record the time once the first response byte is recieved
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb := time.Since(startTime)

			lr.applyResponseTimeUpdate(backendIndex, ttfb)
		},
	}

	return r.WithContext(httptrace.WithClientTrace(r.Context(), trace))
}

func (lr *leastResponse) GetNextBackendIndex(backendList backend.ReadonlyBackendList, r *http.Request) int {
	lowestDuration := time.Duration(math.MaxInt)
	lowestDurationIndex := -1

	for i, duration := range lr.responseTimes {
		if !backendList.Get(i).GetAlive() {
			continue
		}

		if duration < lowestDuration {
			lowestDuration = duration
			lowestDurationIndex = i
		}
	}

	return lowestDurationIndex
}

func (lr *leastResponse) AddBackends(n int) {
	newMeasurementQueues := make([]util.Queue[time.Duration], len(lr.responseTimeMeasurements)+n)
	newRespTimes := make([]time.Duration, len(lr.responseTimes)+n)

	for i := 0; i < len(lr.responseTimeMeasurements); i++ {
		newMeasurementQueues[i] = lr.responseTimeMeasurements[i]
		newRespTimes[i] = lr.responseTimes[i]
	}
	for i := len(lr.responseTimeMeasurements); i < len(lr.responseTimeMeasurements)+n; i++ {
		newMeasurementQueues[i] = util.NewRingBufferQueue[time.Duration](MEASUREMENT_QUEUE_SIE)
		newRespTimes[i] = 0
	}

	lr.responseTimeMeasurements = newMeasurementQueues
	lr.responseTimes = newRespTimes
}

func (lr *leastResponse) RemoveBackends(removedIndices []int) {
	newMeasurementQueues := make([]util.Queue[time.Duration], len(lr.responseTimeMeasurements)-len(removedIndices))
	newRespTimes := make([]time.Duration, len(lr.responseTimes)-len(removedIndices))

	for i, n, m := 0, 0, 0; n < len(newMeasurementQueues); i++ {
		if m >= len(removedIndices) || removedIndices[m] != i {
			// this i was not removed, copy to new and increment new count n
			newMeasurementQueues[n] = lr.responseTimeMeasurements[i]
			newRespTimes[n] = lr.responseTimes[i]
			n++
		} else {
			m++
		}
	}

	lr.responseTimeMeasurements = newMeasurementQueues
	lr.responseTimes = newRespTimes
}
