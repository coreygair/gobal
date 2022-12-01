package strategy

import (
	"errors"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"math"
	"math/rand"
	"net/http"
	"sync"
)

type leastConnections struct {
	connectionCounts     []int
	connectionCountsLock sync.RWMutex
}

func newLeastConnections(cfg config.StrategyConfig, backendManager *backend.BackendManager) *leastConnections {
	return &leastConnections{
		connectionCounts: make([]int, backendManager.GetBackendCount()),
	}
}

func (lc *leastConnections) OnBackendConnectionStart(backendIndex int) {
	lc.connectionCountsLock.Lock()

	lc.connectionCounts[backendIndex]++

	lc.connectionCountsLock.Unlock()
}

func (lc *leastConnections) OnBackendConnectionEnd(backendIndex int) {
	lc.connectionCountsLock.Lock()

	lc.connectionCounts[backendIndex]--

	lc.connectionCountsLock.Unlock()
}

func (lc *leastConnections) GetNextBackendIndex(backendList backend.ReadonlyBackendList, r *http.Request) int {
	// the lowest connection count we have seen
	lowestConnCount := math.MaxInt
	// the number of backends which have lowestConnCount
	numLeastConnBackends := 0
	// list of backends with lowestConnCount [0,numLeastConnBackends)
	leastConnBackendIndexes := make([]int, len(lc.connectionCounts))

	// loop over the backends, checking connection counts
	for i := 0; i < len(lc.connectionCounts); i++ {
		if !backendList.Get(i).GetAlive() {
			continue
		}

		connCount := lc.connectionCounts[i]

		if connCount <= lowestConnCount {

			if connCount != lowestConnCount {
				// connCount < lowestConnCount, so we set lowestConnCount and reset numLeastConnBackends
				lowestConnCount = connCount
				numLeastConnBackends = 0
			}

			// add this backend to the list
			leastConnBackendIndexes[numLeastConnBackends] = i
			numLeastConnBackends++
		}
	}

	if numLeastConnBackends == 0 {
		panic(errors.New("No backends to choose from in least connections."))
	}

	// pick randomly out of the lowest connection backends
	return leastConnBackendIndexes[rand.Intn(numLeastConnBackends)]
}

func (lc *leastConnections) AddBackends(n int) {
	newConnCounts := make([]int, len(lc.connectionCounts)+n)

	for i := 0; i < len(lc.connectionCounts); i++ {
		newConnCounts[i] = lc.connectionCounts[i]
	}
	for i := len(lc.connectionCounts); i < len(lc.connectionCounts)+n; i++ {
		newConnCounts[i] = 0
	}

	lc.connectionCounts = newConnCounts
}

func (lc *leastConnections) RemoveBackends(removedIndices []int) {
	newConnCounts := make([]int, len(lc.connectionCounts)-len(removedIndices))

	for i, n, m := 0, 0, 0; n < len(newConnCounts); i++ {
		if m < len(removedIndices) && i == removedIndices[m] {
			m++
		} else {
			newConnCounts[n] = lc.connectionCounts[i]
			n++
		}
	}

	lc.connectionCounts = newConnCounts
}
