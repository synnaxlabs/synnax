# 20 - Engineering Process Standardization

**Feature Name**: Integration Testing Framework <br />
**Start Date**: 2024-06-18 <br />
**Authors**: Leo Liu<br />
**Status**: Draft <br />

# 0 - Summary

As the Synnax codebase, users, and number of production deployments grow, stability and 
performance of the system become increasingly important characteristics of Synnax to be
measured. In this RFC I propose an in-house framework to allow Synnax to run
easily-configurable and easily-portable integration tests throughout its whole system
(Synnax server, Cesium, Python client, TypeScript client).

# 1 - Motivation

Currently, each component of Synnax has its own unit tests that ensures correctness
for that component. The goal of integration testing is to assure all parts of Synnax
function as expected when run in tandem. This is crucial as real-world use cases do not
utilize any single Synnax component but rather the entire system. A user, for example,
may write data with the Python client while streaming it using the C++ client, and then
deletes from the TypeScript client.

Despite the different parameters and clients, there are only four fundamental operations
available to a user: reading, writing, streaming, and deleting. Thus, all data-related
operations in Cesium is a permutation of these four operations, and integration testing
should allow customization to tests with an equally high degree of freedom.

Recognizing the modular nature of Synnax's operations allows us to simplify the process
of integration test configuration to arranging "blocks" of operations in series or parallel.

# 2 - Design

## 2.1 - Principles

The design of this integration testing framework follows three objectives:
1. Highly configurable: The tester should have control over as many parameters as possible.
This is crucial to understanding the system as a write of 1,000,000 samples may behave
very differently between 1,000 domains of 1,000 samples and 1 domain and 1,000,000 samples.
This means the user should control every parameter, which client to use, how many
goroutines, etc.
2. Highly modular: In the same vein as configurability, each node in the test should be
easily swappable and adding/removing nodes should not involve modification to an
existing test structure. In implementation, this may involve running test nodes in a
client-agnostic manner (i.e. can be run with any client interchangeably.)
3. Well-instrumented: In addition to testing for correctness, the integration test is the
best place to measure Synnax's performance as the performance of unit tests do not
necessarily translate to the performance of the system as a whole.

## 2.2 - Existing tools

The main existing tools considered for this integration framework are golang's `testing`
framework and integration testing tools like k6, Citrus, and Selenium.

I learned valuable lessons from creating the Cesium benchmarking tool: while powerful,
the `testing` tool's benchmark tool cannot help us too much in this task: its forte is
that it can run a test multiple times to stablize timing, but each one of our tests
take a long time, so the benefit of repeatedly running tests may not outweigh the harm
of extra-long testing time. In addition, we have multiple frameworks in multiple
languages, which makes it hard to use one package to run all tests and manage timing –
as we must be careful to not include times such as starting processes, installing
dependencies, etc.

Existing integration testing tools mostly focus on browser testing (i.e. simulating
HTTP requests) – while the Synnax server does run on HTTP requests, this completely bypasses
the client – which is a crucial part of the system. In addition, it is very possible that
the slowest part of the Synnax data pipeline is serializing the data to be transported
over HTTP, so it is absolutely necessary to initiate operations from our clients.

Other open-source tools like Katalon Studio do not allow custom code, however, we want
a high-degree of customization down to the system level (for example, controlling exactly
how many threads to use) and these tools do not satisfy our needs.

Challenge #1: communicate return values, i.e. channel keys, etc.

## 2.3 - Implementation

### 2.3.1 Testing Framework

The design of the integration testing tool is simple: each test is composed of steps, run
sequentially, and each step is composed of nodes, run concurrently on different processes.
To run a test, one needs to simply specify any permutation of Read, Write, Delete, and Stream
operations in nodes and steps in a JSON format, and a go program is used to run the entire
integration test. For example, a test may ressemble this:
```json
"steps": [
    [
        {
            "op": "write",
            "client": "py",
            "params": {
                channels: [...],
                auto_commit: false,
                ...
            }
        },
        {
            "op": "stream",
            "client": "ts",
            "params": {
                ...
            }
        }
    ],
    [
        {
            "op": "write",
            "client": "ts",
            "params": {
                ...
            }
        },
        {
            "op": "delete",
            "client": "ts",
            "params": {
                ...
            }
        },
        {
            "op": "read",
            "client": "py",
            "params": {
                ...
            }
        }
    ]
]
```

In the above example, there are 5 nodes organized in 2 steps, with the first two operations
running in parallel and the the last three running in parallel. Tests written this way
have high customizability and portability, even for when we may have more platforms and
more operations in the future.

In addition to the body, the test suite may also accept `cluster`, `setup`, and `cleanup`
customizations. The `cluster` option allows the test to start a customized cluster (e.g.
TLS, Mem-backed File System); the `setup` option is run before the test steps begin (and
can be useful for things such as creating channels); and the `cleanup` option is run after
the test steps (the only current clean up operation is to delete all channels).

### 2.3.2 Deployment

The integration test is set up as a GitHub action, running on an AWS EC2 instance to
simulate more performant machines that our users may run Synnax on. In addition to testing
that there are no errors in running different parts of the system in tandem, relevant
timing metrics are also provided for each opeartion in the testing report.

## 2.4 Future work

#### Smart closing streamers

Currently, stream operations are only runnable at the same time as write operations, and
they cannot quit autonomously – the tester must manually configure the number of samples
read for the streamer to be closed. Eventually, it would be helpful to close the streamer
once no more data is coming in. Doing so may be challenging, though. We can neither
simply close the streamer once all writers are finished writing, as writers are faster
than streamers, nor can we require streaming exactly the number of samples written by the
writer to close, as streamers may have losses of frames.

#### Channel groups:

Currently, indiviudal channels must be specified by their names to be operated on. This
is painful as writing to 1,000 channels must involve typing 1,000 names in the test
configuration. Ideally, one could use results from past operations as channel groups,
i.e. delete the time range from all channels just written to, read from channels just
deleted from, etc.  

#### Correctness testing:

Currently, the testing framework only asserts that no errors occur when running the
operations. Eventually, it will be equally important to assert that data written and
read are indeed correct.

#### Orphan processes:

When running the cluster with integration tests, we cannot kill the cluster process at the
end – since killing a process does not kill other processes started by it. In addition,
we cannot send a `Interrupt` signal as golang does not implement interrupting a Windows
process (see [issue](https://github.com/golang/go/issues/6720)). Fortunately, the
GitHub Actions runner cleans up orphan processes by default, but this is something that
should be addressed.