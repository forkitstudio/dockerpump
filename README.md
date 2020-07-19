# Intro
dockerpump is free software. You can use it for pump (copy) images from one docker registry to another.

# Features
- Written in Go
- Exposes endpoint JSON RPC API to receive pump commands

# Quick start
## Run the container.
```bash
docker run -it --rm --name="dockerpump" \
    --env DOCKER_REGISTRY_SOURCE_SERVER=some.source.registry.addr:5000 \
    --env DOCKER_REGISTRY_TARGET_SERVER=some.target.registry.addr:5000 \
    -p 10000:10000 forkitstudio/dockerpump
```

## Running in Kubernetes (using a Helm Chart)

Clone the repository (if necessary):
```bash
git clone https://github.com/forkitstudio/dockerpump && cd dockerpump
```
Create namespace and apply the chart:
```bash
kubectl create namespace dp
helm install dockerpump --debug --namespace dp ./helm \
    --set registry.source=<host:port> \
    --set registry.target=<host:port>
```

## API call examples
The following example demonstrates sending a command to copy a busbox image to the target registry: 
```bash
curl -X POST http://127.0.0.1:10000/api/copy_image \
    --header 'Content-Type: application/json' \
    --data-raw '{"repository": "busybox", "tag": "latest"}'
```
The repository_name:tag of the target image will look like: ${DOCKER_REGISTRY_TARGET_SERVER}/busybox:latest

Service healthcheck example:
```bash
curl -X GET http://127.0.0.1:10000/api/health
```

# Environment variables 
* **DOCKER_REGISTRY_SOURCE_SERVER**: sets the address of source Container Registry. If not set, the standard registry will be used.
* **DOCKER_REGISTRY_TARGET_SERVER**: sets the address of target/destination Container Registry. Mandatory.

# Building
```bash
docker build -t dockerpump .
```

# Reference
* https://hub.docker.com/r/forkitstudio/dockerpump
