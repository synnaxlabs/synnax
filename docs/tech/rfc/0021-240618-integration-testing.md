# 21 - Synnax Integration Testing

- **Feature Name**: Integration Testing Framework
- **Start Date**: 2024-06-18
- **Authors**: Leo Liu
- **Status**: Rev 1

# 0 - Summary

As the Synnax codebase, users, and number of production deployments grow, stability and
performance become increasingly important characteristics of Synnax to be measured. In
this RFC I propose an in-house framework to allow Synnax to run easily-configurable and
easily-portable integration tests throughout its whole system (Synnax server, Cesium,
Python client, TypeScript client).

# 1 - Motivation

Currently, each component of Synnax has its own unit tests that ensures correctness for
that component. The goal of integration testing is to assure all parts of Synnax
function as expected when run in tandem. This is crucial as real-world use cases do not
utilize any single Synnax component, but rather the entire system. For example, a user
may write data with the Python client while streaming it using the C++ client, and then
delete from the TypeScript client.

Despite the different parameters and clients, there are only four fundamental operations
available to a user: reading, writing, streaming, and deleting. Thus, all data-related
operations in Cesium is a permutation of these four operations, and integration testing
should allow customization of tests with an equally high degree of freedom.

Recognizing the modular nature of Synnax's operations allows us to simplify the process
of integration test configuration to arranging "blocks" of operations in series or
parallel.

# 2 - Design

## 2.1 - Principles

The design of this integration testing framework follows three objectives:

1. Highly configurable: The tester should have control over as many parameters as
   possible. This is crucial to understanding the system as a write operation consisting
   of 1,000,000 samples may behave very differently depending on whether it is
   configured as 1,000,000 domains of 1 sample each, 1,000 domains of 1,000 samples, or
   1 domain and 1,000,000 samples. This means the user should control every parameter,
   which client to use, how many goroutines, etc.
2. Highly modular: In the same vein as configurability, each node in the test should be
   easily swappable and adding/removing nodes should not involve modification to an
   existing test structure. In implementation, this may involve running test nodes in a
   client-agnostic manner (i.e. can be run with any client interchangeably.)
3. Well-instrumented: In addition to testing for correctness, the integration test is
   the best place to measure Synnax's performance as the performance of unit tests do
   not necessarily translate to the performance of the system as a whole.

## 2.2 - Existing tools

The main existing tools considered for this integration framework are golang's `testing`
framework and integration testing tools like k6, Citrus, and Selenium.

I learned valuable lessons from creating the Cesium benchmarking tool: while powerful,
the `testing` tool's benchmark tool cannot help us too much in this task: its forte is
that it can run a test multiple times to stabilize timing, but each one of our tests
take a long time, so the benefit of repeatedly running tests may not outweigh the harm
of extra-long testing time. In addition, we have multiple frameworks in multiple
languages, which makes it hard to use one package to run all tests and manage timing –
as we must be careful to not include times such as starting processes, installing
dependencies, etc.

Existing integration testing tools mostly focus on browser testing (i.e. simulating HTTP
requests) – while the Synnax server does run on HTTP requests, this completely bypasses
the client – which is a crucial part of the system. In addition, it is very possible
that the slowest part of the Synnax data pipeline is serializing the data to be
transported over HTTP, so it is absolutely necessary to initiate operations from our
clients.

Other open-source tools like Katalon Studio do not allow custom code, however, we want a
high-degree of customization down to the system level (for example, controlling exactly
how many threads to use) and these tools do not satisfy our needs.

Challenge #1: communicate return values, i.e. channel keys, etc.

## 2.3 - Implementation

### 2.3.1 Testing Framework

The design of the integration testing tool is simple: each test is composed of steps,
run sequentially, and each step is composed of nodes, run concurrently on different
processes. To run a test, one needs to simply specify any permutation of Read, Write,
Delete, and Stream operations in nodes and steps in a JSON format, and a go program is
used to run the entire integration test. For example, a test may resemble this:

```json
"steps": [
    [
        {
            "op": "write",
            "client": "py",
            "delay": 0.001,
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
            "delay": 0,
            "params": {
                ...
            }
        },
        {
            "op": "delete",
            "client": "ts",
            "delay": 0,
            "params": {
                ...
                "expected_error": "unauthorized",
                ...
            }
        },
        {
            "op": "read",
            "client": "py",
            "delay": 0,
            "params": {
                ...
                "samples_expected": 10000,
                ...
            }
        }
    ]
]
```

In the above example, there are 5 nodes organized in 2 steps, with the first two
operations running in parallel and the the last three running in parallel. Tests written
this way have high customizability and portability, even for when we may have more
platforms and more operations in the future. One may also add assertions on the number
of samples read during a `read` operation or errors that occur by specifying the
`samples_expected` and/or `expected_error` field. Also note that the `stream` operation
finishes when the streamer has successfully streamed at least 95% of the specified
expected sample count. This is because sometimes streamers never stream all the samples
written due to opening time differences and other factors.

In addition to the body, the test suite may also accept `cluster`, `setup`, and
`cleanup` customizations. The `cluster` option allows the test to start a customized
cluster (e.g. TLS, Mem-backed File System); the `setup` option is run before the test
steps begin (and can be useful for things such as creating channels); and the `cleanup`
option is run after the test steps (the only current clean up operation is to delete all
channels).

Consider the scenario where we want to assert that opening a delete on a channel
currently being written to does indeed produce an `unauthorized` error – simply creating
two parallel nodes of `write` and `delete` will not work, as it is indeterminate which
one runs first – if `delete` runs first, no errors will occur and the test would fail.

Our first attempt at resolving this problem is by creating a property named
`starts_after` – we implemented this using channels that close when a process starts
running and other dependent nodes listening to this channel. However, this did not work
as the order that the commands are started is not necessarily the order that the
operations get run – for example, setting up writers and reading input for writers may
take a longer time than for deletes, resulting in `delete` running before `write` gets
run.

For this reason, we implemented a much less elegant and idiomatic approach – to manually
introduce a time interval after which a command is run to assert the order of execution.
Although unsophisticated, this approach works better than the previous one, as most
operations take at least 10 seconds to run, making timing delays easy. However, one must
still be extremely careful with this manually-introduced delay.

### 2.3.2 Deployment

The integration test is set up as a GitHub action, running on an AWS EC2 instance to
simulate more performant machines that our users may run Synnax on. In addition to
testing that there are no errors in running different parts of the system in tandem,
relevant timing metrics are also provided for each operation in the testing report.

### 2.3.3 Test Cases

Integration testing allows us to test scenarios unachievable in unit tests: concurrent
and cross-OS operations that are very possible in production but hard to achieve in
debugging. To this end, I propose these tests to attempt to expose unexposed errors in
the system:

1. Load / "Everything" test

This is a test that comprises of everything – writing giant amounts of data while trying
to delete, open errant writers on channels in use, etc. The test asserts on disallowing
creating writers on channels already being written to, deleting from channels being
written to, the correctness of streaming, and the correctness of deletes and reads.

The structure of this test is as follows:

```
{
    0-0: write with py
    0-1: write with py
    0-2: write with ts (errant)
    0-3: write with ts
    0-4: delete with py (errant)
    0-5: read with ts (undetermined behavior)
    0-6: stream with ts (assert streaming enough samples)
    0-7: stream with py (assert streaming enough samples)
    1-0: delete with py
    1-1: delete with ts
    2-0: read with py (assert reading correct # of samples)
    2-1: read with ts (assert reading correct # of samples)
}
```

2. Delete integration test

This test asserts that delete is correct with multiple processes and multiple clients
running in parallel. The goal is to test that the correct number of samples remain after
deletion.

```
{
    0-0: write with py
    1-0: delete with ts
    1-1: delete with py
    1-2: delete with ts
    2-0: read with ts
}
```

3. Benchmark comparison test

This test asserts that operations on both clients have the same effect, and serves as a
means to understand the throughput for each operation on the two different clients and
allows for comparison.

## 2.4 Future work

#### Smart-closing streamers

Currently, stream operations are only runnable at the same time as write operations, and
they cannot quit autonomously – the tester must manually configure the number of samples
read for the streamer to be closed. Eventually, it would be helpful to close the
streamer once no more data is coming in. Doing so may be challenging, though. We can
neither simply close the streamer once all writers are finished writing, as writers are
faster than streamers, nor can we require streaming exactly the number of samples
written by the writer to close, as streamers may have losses of frames.

#### Channel groups:

Currently, individual channels must be specified by their names to be operated on. This
is painful as writing to 1,000 channels must involve typing 1,000 names in the test
configuration. Ideally, one could use results from past operations as channel groups,
i.e. delete the time range from all channels just written to, read from channels just
deleted from, etc.

One way to do this, for example, is to write the test configuration file in a format
"smarter" than JSON – for example, a Python List or a Javascript Array. This way, one
can easily refer to the resulting channels of a step. This does not need to abandon all
the code used to parse JSON test configurations, though: to borrow an analogy from the
land of compilers, the tester could write "source code" that get compiled into the
"machine code" that is the JSON configuration file.

#### Correctness testing:

Currently, the testing framework only asserts that no errors occur when running the
operations. Eventually, it will be equally important to assert that data written and
read are indeed correct.

#### Orphan processes:

When running the cluster with integration tests, we cannot interrupt the cluster process
as golang does not implement interrupting a Windows process (see
[issue](https://github.com/golang/go/issues/6720)). Currently, we send a KILL signal to
the server process to forcibly stop it. This is less ideal than a graceful shutdown. In
any case, even though the GitHub Actions runner cleans up orphan processes by default,
but this is something that should be addressed.

As of 07/10/2024, This problem is resolved. The Synnax server is shut down via a
preconfigured stop keyword by entering `"stop"` to `stdin`, circumventing the need to
send an interrupt signal.
