# Traefik Integration Tests

This document describes how to run the integration tests for the Traefik configuration package.

## Prerequisites

- Docker installed
- Go 1.19 or later
- `traefik.yml` in the `./traefik` directory

## Test Setup

1. First, start Traefik using Docker:
```bash
docker run --name traefik-test \
  -p 80:80 \
  -p 8080:8080 \
  --add-host=host.docker.internal:host-gateway \
  -v $PWD/traefik/traefik-integration-test.yml:/etc/traefik/traefik.yml \
  traefik:v3.3 \
  --configfile=/etc/traefik/traefik.yml \
  --log.level=DEBUG \
  --log.format=json
```

2. Set the required environment variables:
```bash
export TRAEFIK_TEST_PORT=9000
```

3. Run the integration tests:
```bash
# Run only integration tests
go test -v -run TestIntegration ./traefik

# Run with debug output
go test -v -run TestIntegration -count=1 ./traefik
```

4. Clean up after tests:
```bash
docker stop traefik-test
docker rm traefik-test
```

## Sample traefik.yml

Here's the required Traefik configuration:
```yaml
# ./traefik/traefik.yml
entryPoints:
  web:
    address: ":80"

providers:
  http:
    endpoint: "http://host.docker.internal:9000/api/config"
    pollInterval: "2s"

api:
  insecure: true  # For testing only
```

Note: `host.docker.internal` is used to access the host machine from within the Docker container. On Linux, you might need to add `--add-host=host.docker.internal:host-gateway` to the Docker run command.

## Troubleshooting

1. If tests fail with connection refused:
   - Verify Traefik is running: `docker ps`
   - Check Traefik logs: `docker logs traefik-test`
   - Ensure ports 80, 9000, and 9001 are available

2. If Traefik can't reach the config server:
   - On Linux, ensure you're using the host.docker.internal DNS
   - Check the config server URL in traefik.yml
   - Verify the TRAEFIK_CONFIG_PORT matches the traefik.yml configuration

3. Common issues:
   ```bash
   # Check if ports are in use
   lsof -i :80
   lsof -i :9000
   lsof -i :9001

   # Check Traefik status
   docker ps -f name=traefik-test
   docker logs traefik-test

   # Verify Traefik can access the config
   curl http://localhost:9000/api/config
   ```

## Complete Test Script

Here's a helper script to run the complete test suite:
```bash
#!/bin/bash

# Stop any existing container
docker stop traefik-test || true
docker rm traefik-test || true

# Start Traefik
docker run -d \
  --name traefik-test \
  -p 80:80 \
  --add-host=host.docker.internal:host-gateway \
  -v $PWD/traefik/traefik.yml:/etc/traefik/traefik.yml \
  traefik:v2.10

# Wait for Traefik to start
sleep 2

# Set environment variables
export TRAEFIK_CONFIG_PORT=9000
export TRAEFIK_ECHO_PORT=9001

# Run tests
go test -v -run TestIntegration ./traefik

# Cleanup
docker stop traefik-test
docker rm traefik-test
```
