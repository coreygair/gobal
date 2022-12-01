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

func TestRoundRobinAdd(t *testing.T) {
	// mock backends list
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("abc", 80),
		config.NewBackendInfo("def", 80),
	})

	cfg := config.StrategyConfig{
		Name: "ROUND_ROBIN",
	}
	rr, _ := newRoundRobin(cfg, bm)

	r, _ := http.NewRequest("GET", "http://abc:80", nil)

	newUrl := []config.BackendInfo{
		config.NewBackendInfo("ghi", 80),
	}
	bm.AddBackends(newUrl)
	rr.AddBackends(len(newUrl))

	indexes := [4]int{-1, -1, -1, -1}
	for i := range indexes {
		indexes[i] = rr.GetNextBackendIndex(bm.GetBackends(), r)
	}

	if indexes != [4]int{0, 1, 2, 0} {
		t.Errorf("Failed correct indices after add: got %v, expected %v.", indexes, [4]int{0, 1, 2, 0})
	}
}

func TestRoundRobinRemove(t *testing.T) {
	// mock backends list
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("abc", 80),
		config.NewBackendInfo("def", 80),
		config.NewBackendInfo("ghi", 80),
		config.NewBackendInfo("jkl", 80),
		config.NewBackendInfo("mno", 80),
	})

	cfg := config.StrategyConfig{
		Name: "ROUND_ROBIN",
	}
	rr, _ := newRoundRobin(cfg, bm)

	r, _ := http.NewRequest("GET", "http://abc:80", nil)

	urls := []config.BackendInfo{
		config.NewBackendInfo("ghi", 80),
		config.NewBackendInfo("mno", 80),
	}
	indicesRemoved := bm.RemoveBackends(urls)
	rr.RemoveBackends(indicesRemoved)

	indexes := [4]int{-1, -1, -1, -1}
	for i := range indexes {
		indexes[i] = rr.GetNextBackendIndex(bm.GetBackends(), r)
	}

	if indexes != [4]int{0, 1, 2, 0} {
		t.Errorf("Failed correct indices after remove: got %v, expected %v.", indexes, [4]int{0, 1, 2, 0})
	}
}

func TestRoundRobinWeighted(t *testing.T) {
	// mock backends list
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("abc", 80),
		config.NewBackendInfo("def", 80),
	})

	cfg := config.StrategyConfig{
		Name: "ROUND_ROBIN_WEIGHT",
		Properties: roundRobinProps{
			Weights: []int{3, 2},
		},
	}
	rr, _ := newRoundRobin(cfg, bm)

	r, _ := http.NewRequest("GET", "http://abc:80", nil)

	fst := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if fst != 0 {
		t.Errorf("First index fail, expected 0 got %d.", fst)
	}

	// respect weights
	snd := rr.GetNextBackendIndex(bm.GetBackends(), r)
	thrd := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if snd != 0 || thrd != 0 {
		t.Errorf("Weights fail, expected [0,0] got [%d,%d].", snd, thrd)
	}

	fourth := rr.GetNextBackendIndex(bm.GetBackends(), r)
	fifth := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if fourth != 1 || fifth != 1 {
		t.Errorf("Increment index fail, expected [1,1] got [%d,%d].", fourth, fifth)
	}

	lst := rr.GetNextBackendIndex(bm.GetBackends(), r)
	if lst != 0 {
		t.Errorf("Loop fail, expected 0 got %d.", lst)
	}
}
