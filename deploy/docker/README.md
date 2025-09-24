# Synnax Deployer

## Quick Start

To build and deploy the Synnax Deployer, along with synnax, run the following commands
in the `deploy` directory:

```bash
docker build -t deployer . && docker run -v /var/run/docker.sock:/var/run/docker.sock -d --name deployer --network=host --restart unless-stopped deployer:latest
```

## Overview

The Synnax Deployer is a Docker container that manages the lifecycle of a synnax
deployment. It is intended to be run alongside the main Synnax Docker container, and is
responsible for keeping the Synnax server up to date.

The Synnax Deployer does the following:

- Periodically checks for updates to the Synnax Docker image by comparing the current
  local image tag with the latest tag on Docker Hub.
- If an update is available, it pulls the latest image and starts polling the Synnax
  server for a successful deployment readiness check.
- Once the check is successful, the Synnax Deployer stops the Synnax container, and
  starts a new container with the latest image.

## Configuration

The Synnax Deployer is configured via editing hard-coded values in the deployment.py
file (this will be changed into environment variables in the future). The following
configuration options are available:

```python
image_name = "synnaxlabs/synnax"  # image name
image_tag = "latest"  # image tag
container_name = "synnax"  # container name
port = "9090"  # host port to expose
container_port = "9090"  # container port to expose
mount_host = "/home/masa/"  # host mount point
mount_container = "/var/"  # container mount point

retry_count = 10  # number of retries to check local container availability
retry_interval = 6  # seconds between retries

poll_interval = 300  # seconds between polls

args = "-vmi"  # container args
```

## Deployment

The Synnax Deployer is intended to be deployed as a Docker container alongside the main
Synnax container. Docker socket access is required for the Synnax Deployer to be able to
manage the external Synnax container.

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -d --name deployer --network=host --restart unless-stopped deployer:latest
```
