package strategy

import (
	"fmt"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"net/http"
	"sync"
)

// Implements the round robin balancing algorithm.
//
// Optionally uses weighted round robin, where each backend is used a number of times before moving on.
type roundRobin struct {
	// Number of backends
	backendCount int

	// The index of the next backend to use
	i int

	// The number of times we have used the current backend
	j int

	// the number of times to use each backend before moving to the next
	weights []int

	// A mutex for reading and writing to this object.
	// Required to guarantee that concurrent invocations read different indexes
	// (rather than one thread reading i before another has written it, therefore both returning the same value)
	m sync.Mutex
}

type roundRobinProps struct {
	Weights []int `yaml:"weights"`
}

// Construct a new round robin strategy from a strategy config, attaching to a backend manager
//
// Returns a ptr to the new strategy, with an error if one occured.
func newRoundRobin(cfg config.StrategyConfig, backendManager *backend.BackendManager) (*roundRobin, error) {
	var props roundRobinProps
	err := config.CastProperties(cfg.Properties, &props)
	if err != nil {
		return nil, fmt.Errorf("Error reading round robin properties: %s", err.Error())
	}

	backendCount := backendManager.GetBackendCount()

	// if there was no weights, use unweighted (1's)
	if props.Weights == nil {
		props.Weights = make([]int, backendCount)
		for i := 0; i < backendCount; i++ {
			props.Weights[i] = 1
		}
	}

	// if weights incomplete, pad with 1's
	if len(props.Weights) < backendCount {
		fmt.Println("Round robin weights too short, padding with 1's.")
		for i := len(props.Weights); i < backendCount; i++ {
			props.Weights[i] = 1
		}
	}

	// if weights too long, truncate
	if len(props.Weights) < backendCount {
		fmt.Println("Round robin weights too long, truncating.")
		props.Weights = props.Weights[:backendCount:backendCount]
	}

	return &roundRobin{
		backendCount: backendCount,
		weights:      props.Weights,
	}, nil
}

func (rr *roundRobin) GetNextBackendIndex(backendList backend.ReadonlyBackendList, r *http.Request) int {
	// aquire the object lock, to avoid 2 concurrent invocations reading the same index
	rr.m.Lock()

	// defer incrementing j,i and unlocking
	defer func() {
		// inc j
		rr.j = rr.j + 1
		// if we have used this backend (j) up to its weight, increment i
		if rr.j >= rr.weights[rr.i] {
			rr.i = (rr.i + 1) % rr.backendCount
			rr.j = 0
		}
		rr.m.Unlock()
	}()

	// prevent infinite looping
	firsti := rr.i

	// ensure we are choosing a 'live' backend
	for !backendList.Get(rr.i).GetAlive() {
		rr.j = 0
		rr.i = (rr.i + 1) % rr.backendCount

		if rr.i == firsti {
			// reached the first i we used, so no live backends
			return -1
		}
	}

	return rr.i
}

func (rr *roundRobin) AddBackends(n int) {
	rr.backendCount += n

	newWeights := make([]int, rr.backendCount)

	copy(newWeights, rr.weights)

	// pad weights with 1's
	for i := len(rr.weights); i < rr.backendCount; i++ {
		newWeights[i] = 1
	}

	rr.weights = newWeights
}

func (rr *roundRobin) RemoveBackends(removedIndices []int) {
	rr.backendCount -= len(removedIndices)

	newWeights := make([]int, len(rr.weights)-len(removedIndices))
	for i, n, r := 0, 0, 0; n < len(newWeights); i++ {
		if r >= len(removedIndices) || i != removedIndices[r] {
			newWeights[n] = rr.weights[i]
			n++
		} else {
			r++
		}
	}

	rr.weights = newWeights
}
