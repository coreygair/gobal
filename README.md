# gobal

A simple load balancer.

## Build

```
make balancer
```

## Demo

Small docker-compose demo. Requires docker and docker-compose.

```
make demo
docker-compose -f demo/docker-compose.yaml up
```

## Usage

```
gobal config_path
```

Starts the balancer based on YAML config in config_path.