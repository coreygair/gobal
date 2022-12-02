package strategy

import (
	"fmt"
	"go-balancer/internal/backend"
	"go-balancer/internal/balancer/config"
	"net/http"
)

// Strategy pattern for choosing the next backend to use.
type BalancerStrategy interface {
	GetNextBackendIndex(backendList backend.ReadonlyBackendList, r *http.Request) int
}

func NewBalancerStrategy(cfg config.StrategyConfig, backendManager *backend.BackendManager) (BalancerStrategy, error) {
	var strat BalancerStrategy
	var err error
	// would love to use a map here but go typing says no :(
	switch cfg.Name {
	case "ROUND_ROBIN":
		fmt.Println("Created round robin balancer.")
		strat, err = newRoundRobin(cfg, backendManager)
		break
	case "LEAST_CONN":
		fmt.Println("Created least connections balancer.")
		strat = newLeastConnections(cfg, backendManager)
		break
	case "LEAST_RESP":
		fmt.Println("Created least response time balancer.")
		strat = newLeastResponse(cfg, backendManager)
		break
	}

	if err != nil {
		return strat, err
	}

	if strat == nil {
		return strat, fmt.Errorf("Unrecognized strategy name '%s' in config file.", cfg.Name)
	}

	// attach strategy methods to backend manager
	var untypedStrat interface{} = strat

	connectionMethods, ok := untypedStrat.(BalancerStrategyConnections)
	if ok {
		fmt.Println("Attaching connection methods.")
		backendManager.ConnectionStartCallback = connectionMethods.OnBackendConnectionStart
		backendManager.ConnectionEndCallback = connectionMethods.OnBackendConnectionEnd
	}

	modifyRequestMethods, ok := untypedStrat.(BalancerStrategyRequestModifier)
	if ok {
		fmt.Println("Attaching request modifier methods.")
		backendManager.ModifyRequestCallback = modifyRequestMethods.ModifyRequest
	}

	return strat, nil
}

// Now, there are some extra interfaces BalancerStrategy implementations can choose to implement to recieve extra information from the backends.

// Recieve notifications about connection events.
type BalancerStrategyConnections interface {
	OnBackendConnectionStart(backendIndex int)
	OnBackendConnectionEnd(backendIndex int)
}

// Modify the incoming requests
type BalancerStrategyRequestModifier interface {
	// Produces a new (or modifies the old) request.
	// Called just before the request is served.
	ModifyRequest(backendIndex int, r *http.Request) *http.Request
}
