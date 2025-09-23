# Technical Foundations

# 0 - Summary

Building real-time distributed systems is no easy task. This document outlines the
software engineering foundations necessary to effectively develop different components
of the Synnax platform.

Different areas of the codebase require different skill sets to work with. Nonetheless,
all of these areas require common foundational knowledge. This guide is organized by
software engineering concept, and, as a way to navigate the subsequent sections, we've
provided a general 'roadmap' for different platform components below.

## 0.0 - Roadmap

### 0.0.0 - Storage Engine

1. [Programming Core in Go](#100---programming-core-in-go)
2. [Essential Abstractions](#11---essential-abstractions)
3. [Building Large Software Systems](#2---building-large-software-systems)
4. [Database Engineering](#4---database-engineering)

### 0.0.1 - Distribution and Networking

1. [Programming Core in Go](#100---programming-core-in-go)
2. [Essential Abstractions](#11---essential-abstractions)
3. [Building Large Software Systems](#2---building-large-software-systems)
4. [Web Services](#5---web-services)
5. [Distributed Systems](#3---distributed-systems)

### 0.0.2 - Core Services

1. [Programming Core in Go](#100---programming-core-in-go)
2. [Essential Abstractions](#11---essential-abstractions)
3. [Web Services](#5---web-services)

### 0.0.3 - Analysis Tooling

1. [Programming Core in Python](#102---programming-core-in-python)
2. [Essential Abstractions](#11---essential-abstractions)

### 0.0.4 - User Interfaces

1. [Programming Core in TypeScript](#103---programming-core-in-javascript-and-typescript)
2. [Package Management - PNPM](#71---pnpm)
3. [Build Systems and Infrastructure - Mono Repo](#80---monorepo)
4. [Build Systems and Infrastructure - Turbo Repo](#81---turbo-repo)
5. [User Interfaces](#9---user-interfaces)
6. [Essential Abstractions](#11---essential-abstractions)
7. [Web Services](#5---web-services)
8. [Building Large Software Systems](#2---building-large-software-systems)

### 0.0.5 - Build Systems and Infrastructure

1. Programming Core in a Language of Choice

# 1 - Philosophy

This guide is **practical**, meaning that it bares little resemblance to a traditional,
theoretical computer science curriculum. Instead, it focuses on the skills that allow
you to **implement** real-world systems. That is not to say that the theoretical is not
relevant. We simply believe that _theory is an emergent property of trying to solve a
problem in practice_; find a real-world problem you want to solve, and learn the theory
you need to solve it.

This guide is **opinionated** in that it focuses specifically on software engineering
for Synnax. This is not to say the content is not directly applicable to other projects,
but rather that we've chosen to omit certain topics that are not relevant to the problem
at hand.

Finally, **we strongly believe that the only way to learn is by doing, and doing a
lot.** Get your hands dirty, make mistakes, and put in the time. A few thousand hours
from now, everything in this guide will seem basic.

# 1 - Programming Core

The first, and most critical, step is to become proficient in the core programming
skills that underlie all work that we do here.

## 1.0 - The Basics

Any free, online programming course should get you quickly through the basics of
programming. Here are the courses we recommend for different programming languages:

### 1.0.0 - Programming Core in Go

#### 1.0.0.0 - Recommended Beginner's Course

If you're new to programming, we recommend starting with the following course:

[Learn Go Programming](https://www.youtube.com/watch?v=YS4e4q9oBaU&ab_channel=freeCodeCamp.org)

#### 1.0.0.1 - Programming in Go for Experienced Programmers

If you're already proficient in another language, we recommend quickly going through the
official go tour:

[A tour of Go](https://go.dev/learn/)

#### 1.0.0.2 - Important Supplements

- The Stack, Heap, and Pointers -
  [Golang pointers explained, once and for all](https://www.youtube.com/watch?v=sTFJtxJXkaY&ab_channel=JunminLee)
- Effective Go - [Effective Go](https://go.dev/doc/effective_go)

### 1.0.2 - Programming Core in Python

#### 1.0.0.0 - Recommended Beginner's Course

If you're new to programming, we recommend starting with
[Learn Python - Full Course for Beginner's](https://www.youtube.com/watch?v=rfscVS0vtbw).

### 1.0.3 - Programming Core in Javascript and TypeScript

#### 1.0.3.0 - Recommended Beginner's Course

If you're new to programming, we recommend starting with
[Learn JavaScript](https://www.youtube.com/watch?v=PkZNo7MFNFg&ab_channel=freeCodeCamp.org)
and supplementing it with
[Learn TypeScript](https://www.youtube.com/watch?v=30LWjhZzg50&ab_channel=freeCodeCamp.org).

#### 1.0.3.1 - Experienced JavaScript Programmer, New to TypeScript

If you're already proficient in JavaScript, we recommend going through the
[TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html).

## 1.1 - Essential Abstractions

After learning the basics, it's time to step into what programming is really about:
abstracting complexity to solve a problem.

### 1.1.0 - Interfaces and Polymorphism

Understanding that software components should be built to satisfy an interface, rather
than provide an implementation, is perhaps the most important realization that enables
engineers to build large, complex systems.

- Implementing interfaces in Python -
  [Protocol or ABC in Python - When to use which one?](https://www.youtube.com/watch?v=xvb5hGLoK0A&t=1s&ab_channel=ArjanCodes).

### 1.1.1 - Classes, Object-Oriented Programming, and Inheritance

### 1.1.2 - Composition

- [The flaws of Inheritance](https://www.youtube.com/watch?v=hxGOiiR9ZKg&t=89s&ab_channel=CodeAesthetic)

- [Python - Why Composition is better than Inheritance](https://www.youtube.com/watch?v=0mcP8ZpUR38&ab_channel=ArjanCodes)

### 1.1.4 - Dependency Injection

After composition, dependency injection and dependency inversion are the most used
structural patterns in the Synnax codebase. Interfaces are foundational in understanding
dependency injection, so make sure you understand them well before moving on. Dependency
injection is particularly powerful in allowing us to test large portions of our code in
a modular fashion.
[This](https://www.youtube.com/watch?v=J1f5b4vcxCQ&t=148s&ab_channel=CodeAesthetic)
video is a great introduction to dependency injection.

### 1.1.3 - Design Patterns

As software engineering evolved from making computations and algorithms work to building
large scale systems, engineers realized that they often encountered problems with
similar properties albeit in different contexts. Design patterns emerged from these
similarities, and they serve as generic, widely applicable solutions to common
challenges. While engineers don't typically apply these in their most pure form,
understanding how to combine and modify these patterns is an essential skill. In Synnax,
we use variations of almost all the common design patterns: Abstract Factory, Builder,
Command, Chain of Responsibility, Iterator, Adapter, Mediator and more.

The quintessential resource on design patterns is
[Design Patterns: Elements of Reusable Object-Oriented Software](https://www.amazon.com/dp/0201633612?ref_=cm_sw_r_cp_ud_dp_56C2VFSRGP5XW20DH7E4).
Written in the 1990's, many consider this as one of the most important books in software
engineering.

Another very useful resource is
[Refactoring Guru - Design Patterns](https://refactoring.guru/design-patterns), which
provides simple, practical examples of design patterns in various programming languages.

### 1.1.4 - The Problems with Abstraction

One of the pitfalls of new/intermediate programmers is to over-abstract. One you become
familiar with a set of patterns, it becomes enticing to apply them in situations where
they may not apply. Fitting a problem into a frame it doesn't fit can end up causing far
more problems than it solves.
[This](https://www.youtube.com/watch?v=rQlMtztiAoA&t=93s&ab_channel=CodeAesthetic) is a
great video explaining how excessive abstraction can be dangerous.

# 2 - Building Large Software Systems

- John Ousterhout's
  [A Philosophy of Software Design](https://www.amazon.com/Philosophy-Software-Design-John-Ousterhout/dp/1732102201)
- John Ousterhout's
  [A Philosophy of Software Design (Lecture)](https://www.youtube.com/watch?v=bmSAYlu0NcY&ab_channel=StanfordUniversitySchoolofEngineering)
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

## 7.0 - Poetry

We use Poetry for managing packages and virtual environments in our Python code. Make
sure you've familiarized yourself with some of the basic commands. Documentation is
available [here](https://python-poetry.org/docs/). As a supplement to this, make sure
you've read our [Python build system guide](./python/python.md).

## 7.1 - PNPM

PNPM is the package manager we use for all of our TypeScript code. Make sure you've
familiarized yourself with some of the basic commands. Documentation is available
[here](https://pnpm.io/). As a supplement to this, make sure you've read our
[frontend build system guide](./typescript/build.md).

# 8 - Build Systems and Infrastructure

## 8.0 - Monorepo

Synnax is organized as a monorepo, meaning that almost all of our code is stored in a
single repository. This is in contrast to a polyrepo, where each package is stored in
its own repository. If you're unfamiliar with what monorepos are and/or what the
benefits they provide, check out
[this video](https://www.youtube.com/watch?v=9iU_IE6vnJ8&t=364s&ab_channel=Fireship).

## 8.1 - Turbo Repo

We use Turbo Repo as our monorepo build system tool for our front end code. It improves
the developer experience by allowing us to build and test multiple packages at once, all
while caching dependencies and build artifacts. Here's a good introduction video called
[turbo repo in 2 minutes](https://www.youtube.com/watch?v=vE3LOHU0OV8&ab_channel=Vercel).
Also, make sure you've familiarized yourself with some of the basic commands.
Documentation is available [here](https://turbo.fyi/).

- Continuous Integration/Continuous Deployment
- Docker
- Kubernetes

# 9 - User Interfaces

# 9.0 - An Introduction to Javascript Frameworks

The world of javascript UI frameworks is one of the most chaotic in software
engineering. Here are two good videos introducing what javascript frameworks are, how to
choose one, and what the current landscape looks like:

- [I built a JavaScript framework](https://www.youtube.com/watch?v=SJeBRW1QQMA&ab_channel=Fireship)
- [JavaScript Frameworks in 2023](https://www.youtube.com/watch?v=S7X6fLbdwlc&ab_channel=Theo-t3%E2%80%A4gg)

Now, accept the fact that for better or wore, we chose React.

# 9.1 - React

We use [React](https://react.dev) as our primary UI framework. It's the most popular
framework available, and is essential to all the frontend work we do. The best way to
get started is to go through the [learn react](https://react.dev/learn) section of the
documentation. We highly recommend going through every section in its entirety, as we
leverage a lot of advanced React patterns in our codebase.

As a supplement, the [React API Reference](https://react.dev/reference/react) is useful
as you start to work your first issues.

- Javascript Frameworks in General
- React
- Redux
- Astro
- GPU Programming
- Tauri

# 9 - Concurrent Programming

- Rob Pike,
  [Concurrency is not Parallelism](https://www.youtube.com/watch?v=oV9rvDllKEg&ab_channel=gnbitcom)
- The Go Blog,
  [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)

# 10 - Profiling

- Profiling in Go

# Useful References

Code Aesthetic - https://www.youtube.com/@CodeAesthetic/videos
