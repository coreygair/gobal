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
