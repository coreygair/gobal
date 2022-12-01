package strategy

import (
	"fmt"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// create a simple server that sleeps for a minute on each request
func createLoopingServer(addr string) http.Server {
	return http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second * 60)
		}),
	}
}

func setupServersAndBackends() ([]http.Server, *backend.BackendManager) {
	// create a few looping servers
	servers := []http.Server{
		createLoopingServer(":9000"),
		createLoopingServer(":9001"),
		createLoopingServer(":9002"),
	}

	// start all the servers on different goroutines
	for _, s := range servers {
		go func(s http.Server) {
			s.ListenAndServe()
		}(s)
	}

	bm := backend.NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("localhost", 9000),
		config.NewBackendInfo("localhost", 9001),
		config.NewBackendInfo("localhost", 9002),
	})

	return servers, bm
}

func teardownServers(servers []http.Server) {
	for _, s := range servers {
		s.Close()
	}
}

func requestBackend(bm *backend.BackendManager, i int) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, bm.GetBackend(i).GetURL().String(), nil)
	go func() {
		err := bm.ServeRequestWithBackend(i, w, r)
		if err != nil {
			fmt.Printf("Err requesting: %s\n", err.Error())
		} else {
			fmt.Println("Done")
		}
	}()
}

func TestLeastConnections(t *testing.T) {
	// seed randomess
	rand.Seed(time.Now().UnixNano())

	servers, bm := setupServersAndBackends()
	defer teardownServers(servers)

	cfg := config.StrategyConfig{
		Name: "LEAST_CONN",
	}
	lc := newLeastConnections(cfg, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	bm.ConnectionStartCallback = lc.OnBackendConnectionStart
	bm.ConnectionEndCallback = lc.OnBackendConnectionEnd

	// test exclude not lowest
	requestBackend(bm, 0)
	// allow connection count to update
	time.Sleep(time.Millisecond * 5)
	for i := 0; i < 5; i++ {
		res := lc.GetNextBackendIndex(bm.GetBackends(), r)
		if res == 0 {
			t.Error("LeastConnections not excluding not lowest.")
		}
	}

	// test pick only lowest
	requestBackend(bm, 2)
	// allow connection count to update
	time.Sleep(time.Millisecond * 5)
	for i := 0; i < 5; i++ {
		res := lc.GetNextBackendIndex(bm.GetBackends(), r)
		if res != 1 {
			t.Errorf("LeastConnections not picking lowest, expected 1 got %d.", res)
		}
	}
}

func TestLeastConnectionsAdd(t *testing.T) {
	// seed randomess
	rand.Seed(time.Now().UnixNano())

	servers, bm := setupServersAndBackends()
	defer teardownServers(servers)

	cfg := config.StrategyConfig{
		Name: "LEAST_CONN",
	}
	lc := newLeastConnections(cfg, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	bm.ConnectionStartCallback = lc.OnBackendConnectionStart
	bm.ConnectionEndCallback = lc.OnBackendConnectionEnd

	requestBackend(bm, 0)
	requestBackend(bm, 1)
	requestBackend(bm, 2)
	// allow connection count to update
	time.Sleep(time.Millisecond * 5)

	newUrls := []config.BackendInfo{config.NewBackendInfo("localhost", 9003)}
	bm.AddBackends(newUrls)
	lc.AddBackends(len(newUrls))

	x := lc.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 3 {
		t.Errorf("Failed get index after add: got %d expected 3", x)
	}
}

func TestLeastConnectionsRemove(t *testing.T) {
	// seed randomess
	rand.Seed(time.Now().UnixNano())

	servers, bm := setupServersAndBackends()
	defer teardownServers(servers)

	cfg := config.StrategyConfig{
		Name: "LEAST_CONN",
	}
	lc := newLeastConnections(cfg, bm)

	r, _ := http.NewRequest("GET", "http://localhost:9000", nil)

	bm.ConnectionStartCallback = lc.OnBackendConnectionStart
	bm.ConnectionEndCallback = lc.OnBackendConnectionEnd

	// request to 2, so that 0 and 1 would be tied
	requestBackend(bm, 2)
	// allow connection count to update
	time.Sleep(time.Millisecond * 5)

	// remove 1, so getnextbackendindex should give 0
	removeURLs := []config.BackendInfo{config.NewBackendInfo("localhost", 9001)}
	removedIndices := bm.RemoveBackends(removeURLs)
	lc.RemoveBackends(removedIndices)

	x := lc.GetNextBackendIndex(bm.GetBackends(), r)
	if x != 0 {
		t.Errorf("Failed get index after remove: got %d expected 0", x)
	}
}
