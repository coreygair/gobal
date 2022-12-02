# gobal

A simple HTTP load balancer.

Includes a few [balancing algorithms](#strategies), common features like sticky sessions, a web monitoring frontend and a [docker-compose demo](#demo).

## Build

```
make balancer
```

Builds a gobal binary in a ```bin``` directory.

## Design

Internally, the load balancer reverse proxies requests to one of several 'backends' (a http address, pointing to a server which can serve the request).

The decision of which backend to use is performed by some balancing strategy, or sticky sessions if turned on.

The balancer also exposes a HTTP API for monitoring backend state as well as adding/removing backends and changing the strategy at runtime.

This is consumed by a webapp the balancer hosts, on localhost port 44444 by default, which provides a simple monitoring frontend. There is also functionality to add/remove backends from here, but this currently does not perform as expected...

## Configuration

gobal is configured at startup by a YAML configuration file of the following format:
```
# Describest the strategy to use
strategy:
    name: STRATEGY_NAME
    properties:
        ...
# A list of backends to use
backends:
    # 'http://{host}:{port}'
    - host: 
      port: 
    ...
# The port for the balancer to listen on
port: 8080
# Use sticky sessions?
sticky: True
```

### Strategies

Here are listed the possible strategy names, and their behaviour

#### ROUND_ROBIN

Implements a simple round robin algorithm, sending one request to each backend in turn.

Can also handle weighted round robin, by passing a list of integer weights under the properties:
```
strategy:
    name: ROUND_ROBIN
    properties:
        weights:
            - 1
            - 2
            - 7
            ...
```

The weights list will be automatically padded with 1's or truncated if it is not the same length as the backend list.

#### LEAST_CONN

Dispatches requests to the backend currently handling the least request.

PLANNED: weighted version.

#### LEAST_RESP

Dispatches rewuests to the backend with the lowest response time to recent requests.

Measured by a simple moving average of recent request TTFB.

### Sticky Sessions

Setting sticky sessions to true allows the balancer to send requests from the same client to the same server each time.

This is currently implemented by setting a simple cookie in responses.

## Usage

```
gobal config_path
```

Starts the balancer based on YAML config in config_path.

## Demo

Small docker-compose demo. Requires docker and docker-compose.

```
make demo
docker-compose -f demo/docker-compose.yaml up
```

Runs a gobal instance and 3 simple website servers. 

The balancer is exposed on the host port 50000, which can then be used to test the balancer behaviour.

Configuration of the demo is specified by [a config file](/demo/balancer/config.yaml), which can be edited to demo different strategies. By default, it uses round robin with sticky sessions enabled.