package backend

import (
	"fmt"
	"go-balancer/internal/balancer/config"
	"net/http"
	"sync"
	"time"
)

type BackendManager struct {
	backends []*backend

	monitor backendMonitor

	ConnectionStartCallback func(backendIndex int)
	ConnectionEndCallback   func(backendIndex int)

	ModifyRequestCallback func(backendIndex int, r *http.Request) *http.Request

	modifyMutex *sync.RWMutex
}

// Creates a new backend manager, with backends from a list of config.BackendInfo's
func NewBackendManager(infos []config.BackendInfo) *BackendManager {
	backends := make([]*backend, len(infos))

	for i, u := range infos {
		backends[i] = newBackend(u)
	}

	bm := &BackendManager{
		backends:    backends,
		modifyMutex: &sync.RWMutex{},
	}

	bm.monitor = backendMonitor{
		// Taking this ptr to the "local" bm variable is ok,
		// as go does escape analysis and will allocate bm on heap :)
		bm:                     bm,
		timeBetweenHeartbeat:   time.Second * 15,
		initialDeadCheckTimer:  time.Second * 5,
		maximumDeadCheckTimer:  time.Second * 10,
		currentDeadCheckTimers: newBackendDurationMap(),
	}

	bm.monitor.StartHeartbeats()

	return bm
}

func (bm *BackendManager) GetBackendCount() int {
	return len(bm.backends)
}

func (bm *BackendManager) GetBackends() ReadonlyBackendList {
	return ReadonlyBackendList{
		list: &bm.backends,
	}
}

func (bm *BackendManager) GetBackend(index int) *backend {
	return bm.backends[index]
}

func (bm *BackendManager) ServeRequestWithBackend(backendIndex int, w http.ResponseWriter, r *http.Request) error {
	bm.modifyMutex.RLock()
	defer bm.modifyMutex.RUnlock()

	if bm.ConnectionStartCallback != nil {
		bm.ConnectionStartCallback(backendIndex)
	}

	if bm.ModifyRequestCallback != nil {
		r = bm.ModifyRequestCallback(backendIndex, r)
	}

	err := bm.backends[backendIndex].serveHTTP(w, r)

	if bm.ConnectionEndCallback != nil {
		bm.ConnectionEndCallback(backendIndex)
	}

	return err
}

// Sets the status of a backend to dead.
// Used when a request to a backend fails, so we want to mark it as dead and not use it in future.
// This also starts a dead checker, periodically testing the backend to see if it comes back up.
//
// Assumes no changes will be made to the backend list between calling and finishing
// (the caller should have already locked the bm)
func (bm *BackendManager) ReportBackendDead(index int) {
	if bm.backends[index].alive {
		bm.backends[index].setAlive(false)
		bm.monitor.BackendDead(bm.backends[index])
	}
}

// Sets the status of a backend to alive.
// Used when a previously dead backend is succesfully accessed by the BackendMonitor.
// If the backend given has been deleted, nothing happens.
//
// Assumes no changes will be made to the backend list between calling and finishing
// (the caller should have already locked the bm)
func (bm *BackendManager) ReportBackendAlive(index int) {
	bm.backends[index].setAlive(true)
}

// Creates new backends and adds them to the list
// Returns an error if a url already exists in a backend
func (bm *BackendManager) AddBackends(infos []config.BackendInfo) error {
	bm.modifyMutex.Lock()
	defer bm.modifyMutex.Unlock()

	newBackends := make([]*backend, len(infos))
	for i, bi := range infos {
		for _, bj := range bm.backends {
			if compareBackendToInfo(bj, &bi) {
				return fmt.Errorf("Error adding backends: url '%s' already exists.", bi.URL.String())
			}
		}
		newBackends[i] = newBackend(bi)
	}

	bm.backends = append(bm.backends, newBackends...)

	return nil
}

// Removes backends by url
// Returns a sorted list of the indexes of backends which were removed (indexes into the list pre-removal)
// No errors: if a url doesnt exist, it is skipped silently
func (bm *BackendManager) RemoveBackends(infos []config.BackendInfo) []int {
	bm.modifyMutex.Lock()
	defer bm.modifyMutex.Unlock()

	removedIndices := make([]int, 0, len(bm.backends))
	nextRemoved, nextKept := 0, 0

	for i, bj := range bm.backends {
		removed := false
		for _, bi := range infos {
			if compareBackendToInfo(bj, &bi) {
				removedIndices = append(removedIndices, i)
				nextRemoved++
				removed = true
				bm.monitor.RemoveBackend(bj)
				break
			}
		}

		if !removed {
			bm.backends[nextKept] = bj
			nextKept++
		}
	}

	bm.backends = bm.backends[:nextKept]

	return removedIndices
}

func compareBackendToInfo(b *backend, info *config.BackendInfo) bool {
	return b.host == info.Host && b.port == info.Port
}
