# Technical Foundations

# 0 - Summary

Building real-time distributed systems is no easy task. This document establishes the
software engineering foundations necessary to effectively develop different components
of the Synnax platform.

Different areas of the codebase require different skill sets to work with. Nonetheless,
all of these share a common core set of skills. As a result, this guide is organized
by software engineering concepts, not by codebase area. As a way to navigate the
subsequent sections, we've provided a general 'roadmap' for different platform components
below.

As a final note, this guide is **practical**, meaning that it bares little resemblance
to a traditional computer science curriculum. Instead, it focuses on the skills that
allow you to **implement** real-world systems. That is not to say that the theoretical
is not important; we simply take the opinion that programming is best learned through
consistent practice.

# 1 - Programming Core

The first, and most critical, step is to learn core programming concepts that apply
universally when developing software.

## 1.0 - The Basics

Any free, online programming course should get you quickly through the basics of
programming. The following is an (exhaustive) list of topics that you should be
familiar with:

1. Variables
2. Control Flow
3. Functions
4. Data Structures - Arrays, Lists, Maps, Sets, Queues, Stacks, Trees, etc.
5. Classes and Basic Object-Oriented Programming
6. Pointers

Here are the courses we recommend for different programming languages:

- [Python](https://www.youtube.com/watch?v=rfscVS0vtbw)
- [JavaScript](https://www.youtube.com/watch?v=PkZNo7MFNFg&ab_channel=freeCodeCamp.org)
  - As a supplement, [TypeScript](https://www.youtube.com/watch?v=30LWjhZzg50&ab_channel=freeCodeCamp.org)
- [Go](https://www.youtube.com/watch?v=YS4e4q9oBaU&ab_channel=freeCodeCamp.org)


## 1.1 - Essential Abstractions

1. Interfaces and Polymorphism
2. Classes, Object-Oriented Programming, and Inheritance
3. Composition, and why we prefer it over Inheritance
4. Design Patterns

- [Design Patterns: Elements of Reusable Object-Oriented Software](https://www.amazon.com/dp/0201633612?ref_=cm_sw_r_cp_ud_dp_56C2VFSRGP5XW20DH7E4)

# 2 - Building Large Software Systems

- John Ousterhout's [A Philosophy of Software Design](https://www.amazon.com/Philosophy-Software-Design-John-Ousterhout/dp/1732102201)
- Martin Kleppmann's [Designing Data-Intensive Applications](https://a.co/d/4rHgKH3)

# 3 - Distributed Systems

- Martin Van Steen, [Distributed Systems](https://a.co/d/017uaCQ)

# 4 - Database Engineering

- Alex Petrov, [Database Internals](https://a.co/d/jfIHa0D)

# 5 - Web Services

- The IP Suite
- HTTP
- TCP/UDP
- TLS
- Mutual TLS

# 6 - Data Analysis

- [Data Analysis with Python](https://www.youtube.com/watch?v=r-uOLxNrNk8&ab_channel=freeCodeCamp.org)

# 7 - Package Management

- Poetry
- PNPM

# 8 - Build Systems and Infrastructure

- Continuous Integration/Continuous Deployment
- Docker
- Kubernetes
- Turborepo

# 9 - User Interfaces

- React
- Redux
- Astro
- GPU Programming

# 9 - Concurrent Programming

- Rob Pike, [Concurrency is not Parallelism](https://www.youtube.com/watch?v=oV9rvDllKEg&ab_channel=gnbitcom)

# 10 - Profiling

- Profiling in Go
