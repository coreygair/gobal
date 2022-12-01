package strategy

import (
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"net/http"
	"testing"
	"time"
)

func TestLeastResponse(t *testing.T) {
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("localhost", 9000),
		config.NewBackendInfo("localhost", 9001),
	})

	lr := newLeastResponse(config.StrategyConfig{
		Name: "LEAST_RESP",
	}, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	// populate fake response times (hard to test real ones)
	// only 1 result each, so will be directly used
	lr.applyResponseTimeUpdate(0, time.Millisecond*500)
	lr.applyResponseTimeUpdate(1, time.Millisecond*100)

	x := lr.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 1 {
		t.Errorf("Failed get lowest response time: got %d, expected 1", x)
	}

	// apply a longer measurement to 1
	// avg of 100 and 1000 is 550, which is bigger than 500 for 0
	lr.applyResponseTimeUpdate(1, time.Millisecond*1000)

	x = lr.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 0 {
		t.Errorf("Failed get lowest response time after average update: got %d, expected 0", x)
	}

	// clear measurements
	lr = newLeastResponse(config.StrategyConfig{
		Name: "LEAST_RESP",
	}, bm)

	bm.ModifyRequestCallback = lr.ModifyRequest

	// this time, testing if after 10 updates we forget oldest
	// done by starting with a massive time and then adding some smaller ones until it changes
	lr.applyResponseTimeUpdate(0, time.Millisecond*100)
	lr.applyResponseTimeUpdate(1, time.Hour*100)

	// add 9 small times, but the avg shouldnt fall under 100 microsec
	for i := 0; i < 9; i++ {
		lr.applyResponseTimeUpdate(1, time.Millisecond*10)
		x = lr.GetNextBackendIndex(bm.GetBackends(), r)
		if x != 0 {
			t.Errorf("Failed average converges too fast: got %d, expected 0", x)
		}
	}

	// now we should forget the oldest
	lr.applyResponseTimeUpdate(1, time.Millisecond*10)
	x = lr.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 1 {
		t.Errorf("Failed forget old measurements: got %d, expected 1", x)
	}
}

func TestLeastResponseAdd(t *testing.T) {
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("localhost", 9000),
		config.NewBackendInfo("localhost", 9001),
	})

	lr := newLeastResponse(config.StrategyConfig{
		Name: "LEAST_RESP",
	}, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	bm.ModifyRequestCallback = lr.ModifyRequest

	newUrls := []config.BackendInfo{
		config.NewBackendInfo("localhost", 9003),
	}
	bm.AddBackends(newUrls)
	lr.AddBackends(len(newUrls))

	lr.applyResponseTimeUpdate(0, 10)
	lr.applyResponseTimeUpdate(1, 10)
	lr.applyResponseTimeUpdate(2, 5)

	x := lr.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 2 {
		t.Errorf("Failed get index after add: got %d expected 2", x)
	}
}

func TestLeastResponseRemove(t *testing.T) {
	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("localhost", 9000),
		config.NewBackendInfo("localhost", 9001),
		config.NewBackendInfo("localhost", 9002),
		config.NewBackendInfo("localhost", 9003),
		config.NewBackendInfo("localhost", 9004),
	})

	lr := newLeastResponse(config.StrategyConfig{
		Name: "LEAST_RESP",
	}, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	bm.ModifyRequestCallback = lr.ModifyRequest

	removeUrls := []config.BackendInfo{
		config.NewBackendInfo("localhost", 9002),
		config.NewBackendInfo("localhost", 9004),
	}
	indices := bm.RemoveBackends(removeUrls)
	lr.RemoveBackends(indices)

	lr.applyResponseTimeUpdate(0, 10)
	lr.applyResponseTimeUpdate(1, 10)
	lr.applyResponseTimeUpdate(2, 5)

	x := lr.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 2 {
		t.Errorf("Failed get index after add: got %d expected 2", x)
	}
}
