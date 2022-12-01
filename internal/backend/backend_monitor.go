package backend

import (
	"fmt"
	"hash/maphash"
	"net/http"
	"time"
)

type backendDurationMap struct {
	m map[uint64]time.Duration

	hashSeed maphash.Seed
}

func newBackendDurationMap() backendDurationMap {
	return backendDurationMap{
		m:        make(map[uint64]time.Duration),
		hashSeed: maphash.MakeSeed(),
	}
}

func (bdm *backendDurationMap) Set(b *backend, d time.Duration) {
	bdm.m[maphash.String(bdm.hashSeed, b.url.String())] = d
}

func (bdm *backendDurationMap) Get(b *backend) (time.Duration, bool) {
	d, ok := bdm.m[maphash.String(bdm.hashSeed, b.url.String())]
	return d, ok
}

func (bdm *backendDurationMap) Delete(b *backend) {
	delete(bdm.m, maphash.String(bdm.hashSeed, b.url.String()))
}

// Performs regular heartbeat tests to ensure backends are alive.
// Also, performs dead checks on reported dead servers to check if they come back.
type backendMonitor struct {
	// The backend manager on which to test backends.
	bm *BackendManager

	// Time between each heartbeat to alive backends.
	timeBetweenHeartbeat time.Duration

	// The time between the first dead checks.
	initialDeadCheckTimer time.Duration
	// The maximum time between dead checks.
	maximumDeadCheckTimer time.Duration

	// Map from backend to current dead check duration.
	currentDeadCheckTimers backendDurationMap
}

// Starts regularly heartbeat testing each backend.
func (monitor *backendMonitor) StartHeartbeats() {
	fmt.Println("Started heartbeats.")

	go func() {
		for true {
			time.Sleep(monitor.timeBetweenHeartbeat)

			fmt.Println("Heartbeat")
			monitor.performHeartbeats()
		}
	}()
}

// Loops over every current backend and polls it to ensure it is alive.
func (monitor *backendMonitor) performHeartbeats() {
	// Have to aquire a lock for reading from the bm, as this could take some time
	// During which someone might try to make changes to the backend list
	monitor.bm.modifyMutex.RLock()
	defer monitor.bm.modifyMutex.RUnlock()

	backends := monitor.bm.GetBackends()

	for i := 0; i < backends.Len(); i++ {
		b := backends.Get(i)

		// skip dead backends
		if !b.GetAlive() {
			continue
		}

		fmt.Printf("Heartbeating %s\n", b.url.String())
		_, err := http.Head(b.url.String())
		if err != nil {
			fmt.Println("dead")
			monitor.bm.ReportBackendDead(backends.IndexOf(b))
		}
	}
}

func (monitor *backendMonitor) RemoveBackend(b *backend) {
	monitor.currentDeadCheckTimers.Delete(b)
}

func (monitor *backendMonitor) BackendDead(b *backend) {
	_, present := monitor.currentDeadCheckTimers.Get(b)
	if present {
		// already a checker up for this, don't bother
		return
	}

	// start dead checker
	go monitor.deadChecker(b)
}

// Periodically checks the liveness of a previously dead backend.
// Exponentially backs off checking to a maximum duration.
func (monitor *backendMonitor) deadChecker(b *backend) {
	monitor.currentDeadCheckTimers.Set(b, monitor.initialDeadCheckTimer)

	for true {
		dur, present := monitor.currentDeadCheckTimers.Get(b)
		if !present {
			// the backend was deleted at some point, so stop checking it
			return
		}

		// sleep for the current sleep duration
		time.Sleep(dur)

		fmt.Printf("Dead checking %s\n", b.url.String())

		// Lock bm to read and find the backends index
		monitor.bm.modifyMutex.RLock()

		// now, check if the backend is up
		_, err := http.Head(b.url.String())

		if err == nil {
			// back up!
			fmt.Println("up!")
			monitor.currentDeadCheckTimers.Delete(b)
			monitor.bm.ReportBackendAlive(monitor.bm.GetBackends().IndexOf(b))

			monitor.bm.modifyMutex.RUnlock()

			return
		}

		// increase wait time
		newDur := dur * 2
		if newDur > monitor.maximumDeadCheckTimer {
			newDur = monitor.maximumDeadCheckTimer
		}
		monitor.currentDeadCheckTimers.Set(b, newDur)

		// Release the bm mutex
		monitor.bm.modifyMutex.RUnlock()
	}
}
