package strategy

import (
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"net/http"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	// mock backends list
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("abc", 80),
		config.NewBackendInfo("def", 80),
		config.NewBackendInfo("ghi", 80),
	})

	cfg := config.StrategyConfig{
		Name: "ROUND_ROBIN",
	}
	rr, _ := newRoundRobin(cfg, bm)

	r, _ := http.NewRequest("GET", "http://abc:80", nil)

	fst := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if fst != 0 {
		t.Errorf("First index fail, expected 0 got %d.", fst)
	}

	snd := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if fst != 0 {
		t.Errorf("Increment index fail, expected 1 got %d.", snd)
	}

	rr.GetNextBackendIndex(bm.GetBackends(), r)
	lst := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if fst != 0 {
		t.Errorf("Loop index fail, expected 0 got %d.", lst)
	}
}
