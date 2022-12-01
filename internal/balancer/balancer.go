package balancer

import (
	"fmt"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"go-balancer/internal/balancer/strategy"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type balancer struct {
	backendManager *backend.BackendManager
	strategy       strategy.BalancerStrategy
	sticky         bool

	modifyMutex sync.RWMutex
}

func NewBalancer(cfg config.Config) (balancer, error) {
	bm := backend.NewBackendManager(cfg.Backends)

	strategy, err := strategy.NewBalancerStrategy(cfg.Strategy, bm)
	if err != nil {
		return balancer{}, err
	}

	return balancer{
		backendManager: bm,
		strategy:       strategy,
		sticky:         cfg.Sticky,
	}, nil
}

// Change the strategy the current balancer is using
// Takes a new config.StrategyConfig describing the new strategy
// Returns an error if using the config to instantiate a strategy failed
func (b *balancer) ChangeStrategy(newStrategyCfg config.StrategyConfig) error {
	b.modifyMutex.Lock()
	defer b.modifyMutex.Unlock()

	// keep hold of the old strategy for recovery
	oldStrategy := b.strategy

	// have to do this as go doesnt like the initialiser syntax with b.strategy
	var err error
	b.strategy, err = strategy.NewBalancerStrategy(newStrategyCfg, b.backendManager)
	if err != nil {
		// if the new strategy is invalid, recover the old one
		b.strategy = oldStrategy
	}
	return err
}

// Handles adding backends by BackendInfo to the balancer
//
// Aquires a mutex which prevents other actions happening on the balancer,
// before notifying the balancers components of the new backends
func (b *balancer) AddBackends(infos []config.BackendInfo) error {
	b.modifyMutex.Lock()
	defer b.modifyMutex.Unlock()

	err := b.backendManager.AddBackends(infos)
	if err != nil {
		return err
	}

	b.strategy.AddBackends(len(infos))

	return nil
}

// Handles removing backends by url
//
// Aquires a mutex which prevents other actions happening on the balancer,
// before notifying the balancers components of the removed backends
func (b *balancer) RemoveBackends(infos []config.BackendInfo) {
	b.modifyMutex.Lock()
	defer b.modifyMutex.Unlock()

	removedIndices := b.backendManager.RemoveBackends(infos)

	b.strategy.RemoveBackends(removedIndices)
}

// Serve a http request using the balancer.
//
// Selects an appropriate backend and reverse proxies the request to it.
func (b *balancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// get a read lock for this balancer
	// ensures there is no modification happening to this for the duration of the request
	b.modifyMutex.RLock()
	defer b.modifyMutex.RUnlock()

	// note no need to read lock the backend manager, as this is the only object that will ever 'write' to it
	// this is locked by the above lock, so we are safe to use backend manager knowing the backend list wont change

	if b.sticky {
		// check for balancer session
		cookie, err := r.Cookie(balancerSessionCookieName)

		if err == nil {
			// there is a session cookie
			backendIndex, err := strconv.ParseInt(cookie.Value, 0, 0)

			if err != nil {
				fmt.Printf("Error parsing session cookie: %s", err)
			} else {
				// we have a session, use that backend
				err := b.backendManager.ServeRequestWithBackend(int(backendIndex), w, r)

				if err != nil {
					// error with sessioned server, fall through to balancing strat
					b.backendManager.ReportBackendDead(int(backendIndex))
				} else {
					// request served, refresh cookie and exit
					setBalancerSessionCokie(w, int(backendIndex))
					return
				}
			}
		}
	}

	success := false
	for i := 0; i < 3; i++ {
		backends := b.backendManager.GetBackends()
		backendIndex := b.strategy.GetNextBackendIndex(backends, r)

		if backendIndex == -1 {
			// no available backends
			w.WriteHeader(http.StatusBadGateway)

			fmt.Println("No available backends to service request!")

			return
		}

		// add cookie to resp (must do this before req is served)
		// if the backend fails, doesnt matter as will replace on retry
		if b.sticky {
			setBalancerSessionCokie(w, backendIndex)
		}

		// Serve the request with the backends reverse proxy
		err := b.backendManager.ServeRequestWithBackend(backendIndex, w, r)

		if err != nil {
			// the backend produced an error, so report it as dead
			b.backendManager.ReportBackendDead(backendIndex)

			// log error
			fmt.Printf("Error using backend '%s': %s\n", backends.Get(backendIndex).GetURL().String(), err.Error())
		} else {
			success = true
			break
		}
	}

	if !success {
		// if we ran out of retries, failed
		setBalancerDeleteSessionCookie(w)
		w.WriteHeader(http.StatusBadGateway)
	}
}

const balancerSessionCookieName = "balancer_session"
const balancerSessionCookieLifetime = time.Minute * 15

func setBalancerSessionCokie(w http.ResponseWriter, backendIndex int) {
	http.SetCookie(w, &http.Cookie{
		Name:    balancerSessionCookieName,
		Value:   fmt.Sprintf("%d", backendIndex),
		Expires: time.Now().Add(balancerSessionCookieLifetime),
	})
}
func setBalancerDeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   balancerSessionCookieName,
		MaxAge: 0,
	})
}
