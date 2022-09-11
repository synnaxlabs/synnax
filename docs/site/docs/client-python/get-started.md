---
sidebar_position: 1
---

# Get Started

Arya offers a Python client for communicating with a cluster. 

## Installation

The Python client is available on PyPI, and can be installed using pip:

```bash
pip install arya-client
```

## Connecting to a Cluster

To connect to a cluster, import the `Client` class and instantiate it with the host and port of a reachable node in
the cluster:

```python
import arya

client = arya.Client(host="localhost",port=8080)
```

## Write Telemetry

To write telemetry to a cluster, 


