package strategy

import (
	"fmt"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"go-balancer/internal/hashing"
	"net/http"
)

type requestHash struct {
	ring hashing.ConsistentHash[backend.BackendRef]

	requestHasher func(*http.Request) uint64
}

type requestHashProps struct {
	DuplicationFactor int `yaml:"duplicationFactor"`
}

func newRequestHash(cfg config.StrategyConfig, backendManager *backend.BackendManager) (*requestHash, error) {
	var props requestHashProps
	err := config.CastProperties(cfg.Properties, &props)
	if err != nil {
		return nil, fmt.Errorf("Error reading request hash properties: %s", err.Error())
	}

	ch := hashing.NewConsistentHash[backend.BackendRef](func(b backend.BackendRef) string {
		return b.GetURL().String()
	})

	backends := backendManager.GetBackends()
	for i := 0; i < backends.Len(); i++ {
		ch.Add(backends.Get(i), props.DuplicationFactor)
	}

	return &requestHash{
		ring: ch,
		// TODO: change based on props
		requestHasher: func(r *http.Request) uint64 {
			return 1
		},
	}, nil
}

func (h *requestHash) GetNextBackendIndex(backendList backend.ReadonlyBackendList, r *http.Request) int {
	hashed := h.requestHasher(r)

	return backendList.IndexOf(h.ring.RingLookup(hashed))
}

func (h *requestHash) AddBackends(n int) {

}

func (h *requestHash) RemoveBackends(indexes []int) {

}
