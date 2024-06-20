# 20 - Engineering Process Standardization

**Feature Name**: Integration Testing Framework <br />
**Start Date**: 2024-06-18 <br />
**Authors**: Leo Liu<br />
**Status**: Draft <br />

# 0 - Summary

As the Synnax codebase, users, and number of production deployments grow, stability and 
performance of the system become increasingly important characteristics of Synnax to be
measured. In this RFC I propose a framework to allow Synnax to run easily-configurable
and easily-portable integration tests throughout its whole system (Synnax server, Python
client, C++ client, TypeScript client, Driver).

# 1 - Motivation

Currently, each individual component of Synnax has its own unit tests that ensures correctness
for that small component. The goal of integration testing is to assure all parts of Synnax
function as expected when run in tandem – this simulates real-world use case scenarios where
a user, for example, writes data with the C++ client (via the driver) while streaming it using
the Python client, and then decides to delete from the TypeScript client. We can see that
despite the different parameters and different clients, there are only four fundamental operations
that a user uses: reading, writing, streaming, and deleting. Thus, all forms of integration testing
will involve a permutation of these four operations, in some order.

This allows us the abstract the actual process and details of the operation from the user: apart from
what client a write uses and how much data to write, each write is the same as the other – therefore,
we can simplify the process of test configuration to arranging "blocks" of operations in series or parallel.

Lessons learned from Cesium benchmarking: while powerful, the go test bench tool cannot help us too much: its
forte is that it can run a test multiple times to average out timing, but each one of our tests take a long time.
Also, we must set up each test so the DB state is immutable: this is not suitable for us. In addition, we eventually
want to run integration tests on the cloud – which could resemble deployment scenarios. Lastly, we have multiple
frameworks in multiple languages, this makes it hard to use one existing framework to unify.

Research into existing tools:
- Most tools like k6, Citrus, Selenium only support browser testing (i.e. via HTTP requests) – we want to actually use our client
 and configure more complex / concurrent tests – also, serialization time must be taken into account: it is one of the biggest
 time killers of Synnax.
- Other open-source tools like Katalon Studio does not allow custom code only predefined options – so while it's doable, not
very customizable
- We want to have control over how many threads to use, etc.

Challenge #1: communicate return values, i.e. channel keys, etc.