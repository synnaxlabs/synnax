# 13 - Pluto Visualization

- **Feature Name**: Pluto Visualization
- **Start Date**: 2023-05-30
- **Authors**: Emiliano Bonilla
- **Status**: Released

# 0 - Summary

Usable, high performance data visualization is at the core of what Synnax is offering.
The implementation of [telemetry streaming](./0012-230501-telemetry-streaming.md)
demands a significant change to how we approach the data fetching and rendering process.
The current design is also highly monolithic and tightly coupled. In this RFC I propose
a new architecture for Synnax's visualization system, implementing a modular component
based framework that shifts the responsibility of data fetching and rendering completely
off of the main thread. The goal is to keep the user facing API 'reacty' while
leveraging all the benefits of moving heavy lifting to a worker thread.

# 1 - Vocabulary

- **Pluto** - The Synnax React component library. Source code available
  [here](../../../pluto).
- **Telemetry** - Data samples received from sensors and sent to actuators; typically
  stored on Synnax server. More details available [here](../../../pluto).
- **Series** - A strongly typed collection of telemetry samples over a time range. The
  fundamental unit of data transfer in Synnax server.

# 2 - Motivation

Line based visualization is by far the primary means of accessing the data that Synnax
stores. Many existing visualization systems feel clunky, and are typically intended for
static, small data sets. By providing an interface that allows access and exploration of
large, live data sets, we're empowering our users to take advantage of all the advanced
tooling Synnax has to offer, ultimately delivering our users a much

The current rendering pipeline requires reloading the entire data set on every update.
Now that we'll be updating at rates of 100Hz or more, this approach is no longer
sustainable. We need a method that only receives new data, and only renders the areas of
the canvas that have changed.

It also turns out that rendering lines and axes on a screen is remarkably complex,
requiring many coordinate transformations, GPU memory management, caching mechanisms,
and more. The current design decisions were made when we had far less knowledge of the
problem space and resulted in unorganized, tightly coupled code. We need a design that
provides clear isolation between different areas of the rendering process, allowing us
to make incremental improvements to each area without affecting others.

# 3 - Philosophy

## 3.0 - React Remains in Control

Pluto is a React component library, and maintaining a react-focused API lifts the burden
of implementing custom logic to decide the structure and lifecycle of visualization
components. A React-focused API is also more familiar to our users, allowing them to use
high-powered multithreaded visualizations as if they were working with simple
components.

## 3.1 - Offload the Heavy Lifting to a Worker Thread

The bulk of Pluto's memory usage and computation comes from fetching and rendering
buffers of telemetry. Common operations include counting minimums and maximums,
converting data types, and buffering values to the GPU. In many cases, these operations
can be so intense that they can block the main thread and prevent any user interactions.
As we add live telemetry, many of these processing operations aren't going to be user
driven, but will instead be triggered by incoming data from the server. By shifting
these expensive operations to a worker process, we can drastically reduce the compute
load on the main thread, leading to a much smoother experience for our users even when
working with large, real-time data sets.

## 3.2 - The Visualization Core Remains Generic

Pluto is a component library, and as such, the core visualization logic should remain
independent of any Synnax specific interfaces/telemetry systems. This provides us with a
clear separation of concerns that guarantees that as the Synnax Data API inevitably
changes, the way we visualize that data remains the same (and vice versa). A consequent
benefit is that we can easily test the visualization core in isolation, without needing
to have a database spun up.

# 5 - Detailed Design

## 5.0 - The Aether Component Framework

The central pattern for implementing Pluto's visualization system is the Aether
component framework. Aether implements a composite component tree on a worker thread
that mirrors the React tree on the main thread. Using the `Aether.use` hook, a React
component can create a new component on the worker thread and share state with it. The
worker component can also modify the shared state, allowing it to communicate back to
the main thread.

### 5.0.0 - The Aether Component Tree

On the worker thread, Aether maintains a composite tree of components whose
implementation feels similar to a Class-based React component. To fork a new component,
we use the `render` function, which takes in a registry of component factories. The
`render` function then receives messages from the main thread to update the component
tree, creating and destroying components as necessary.

```typescript
// worker.ts

import { Aether, AetherComponentRegistry } from "@synnaxlabs/pluto";
import { MyWorkerButton } from "./MyWorkerButton";
import { MyWorkerLinePlot } from "./MyWorkerLinePlot";

const REGISTRY: AetherComponentRegistry = {
  [MyWorkerButton.TYPE]: MyWorkerButton,
  [MyWorkerLinePlot.TYPE]: MyWorkerLinePlot,
};

Aether.render(REGISTRY);
```

There are two types of component that can be created in the Aether component tree:
`Composite` and `Leaf`. A `Composite` component can have children, while a `Leaf`
component cannot. To create a `Composite` component, we extend the `AetherComposite`
class, and, likewise, to create a `Leaf` component, we extend the `AetherLeaf` class:

```typescript
// MyWorkerButton.ts

import { AetherLeaf } from "@synnaxlabs/pluto";

export class MyWorkerButton extends AetherLeaf {
  static TYPE = "MyWorkerButton";

  constructor(initialState) {
    super(initialState);
  }

  handleUpdate() {
    console.log("I'm doing something with state", this.state);
  }
}
```

It's important to note that the subclass we implement for an `AetherComposite` does
**not** have control over the lifecycle of its children (can't create them, delete them,
or set their state). This is intentional, as the aether component tree is driven by
React on the main thread. The worker component tree does, however, have access to its
children, and can execute methods on them.

**Aether does not implement any rendering patterns**. All aether does is maintain a tree
of stateful components and allow the user to respond to state changes. In some cases, a
component may want to render something to the screen, but, in other cases, the component
may only be used for computation or data fetching.

### 5.0.1 - The Need for Context

Pluto makes extensive use of React's context API to provide components with access to
important tooling. The most notable example here is the visualization canvas, which
provides a WebGL rendering context to all components that need it. Now that we're moving
the core functionality of many components off of the main thread, we have a plethora of
contextual information that needs to be accessed by them.

To solve this problem, Aether implements a very rudimentary context API. When a
component updates, it receives an `AetherContext` object that contains a map of
arbitrary key-value pairs. After a component receives a state update, Aether checks if
the component has modified the context map. If so, Aether _unconditionally_ updates all
the component's children, allowing them to alter their state based on the context
changes. Obviously, this is a naive and inefficient approach, but implementing a robust
context API requires considerable effort, and this approach is sufficient for our needs.

### 5.0.2 - Issues with `React.StrictMode`

React's `StrictMode` forcibly re-renders a component twice and runs its effects twice.
Aether uses an ID generator (`nanoid`) to assign unique keys to components. Without
strict mode, the hooks that manage the lifecycle of the component work as expected. Even
in the case of effects running multiple times, the components works. The problem arises
because react renders the component one twice _and then_ runs the effects twice. This
means that the initial, synchronous bootstrapping code for a component runs for the
first rerender but never gets cleaned up. This is intentional behavior by the React
team, and is useful for catching bugs, but in our situation it's a problem.

## 5.0 - Visualization Component Structure

The most challenging roadblock with the previous visualization architecture was the
large, tightly coupled, and very complex functions and classes that handled the assembly
and drawing process. The separation of concerns was remarkably unclear, and refactoring
and adding features was remarkably challenging.

The new architecture separates these concerns by leveraging composition using Aether.
The gist is to present a category of visualization as a container component (i.e.
`LinePlot`, `Valve`, or `Table`) and then allow the user to customize its layout using
children.

### 5.0.0 - The Line Plot Component

To demonstrate the flexibility of this approach, we'll use the line plot component as an
example. The code for a plot a single line is as follows:

```typescript jsx
<LinePlot>
    <LinePlot.XAxis label="Time" variant="time" location="bottom">
        <LinePlot.YAxis label="Pressure" variant="pressure" location="left">
            <LinePlot.Line telem={pressureTelemetry}/>
        </LinePlot.YAxis>
    </LinePlot.XAxis>
</LinePlot>
```

The structure here is self-explanatory. Just by looking, we know we have a single line
whose data comes from the source `pressureTelemetry` and is scaled to the Y and X axes
that are its parents. Adding another line is simple:

```typescript jsx
<LinePlot>
    <LinePlot.XAxis label="Time" variant="time" location="bottom">
        <LinePlot.YAxis label="Pressure" variant="pressure" location="left">
            <LinePlot.Line telem={pressureTelemetry}/>
        </LinePlot.YAxis>
        <LinePlot.YAxis label="Temperature" variant="temperature" location="right">
            <LinePlot.Line telem={temperatureTelemetry}/>
        </LinePlot.YAxis>
    </LinePlot.XAxis>
</LinePlot>
```

Now we've added another Y axis to hold our line for temperature data. Both of these
lines share the same X axis. We can also introduce a title to the plot as follows:

```typescript jsx
<LinePlot>
    <LinePlot.Title>My Line Plot</LinePlot.Title>
    <LinePlot.XAxis label="Time" variant="time" location="bottom">
        <LinePlot.YAxis label="Pressure" variant="pressure" location="left">
            <LinePlot.Line telem={pressureTelemetry}/>
        </LinePlot.YAxis>
        <LinePlot.YAxis label="Temperature" variant="temperature" location="right">
            <LinePlot.Line telem={temperatureTelemetry}/>
        </LinePlot.YAxis>
    </LinePlot.XAxis>
</LinePlot>
```

It's easy to imagine how this pattern can be extended to add annotations, tooltips,
additional axes, and more. This approach is extremely intuitive from a DX perspective,
and also gives us a clear method for separating concerns within the implementation.

## 5.1 - Integrating Telemetry Sources

Correctly integrating telemetry sources into the visualization component structure is a
challenge. The goal is to allow users to intuitively define telemetry sources on the
main thread and then have them automatically linked to the corresponding component in
the worker thread.

The largest hurdle here is that the client-side telemetry infrastructure (clients,
sockets, caches) are _stateful_. We need to maintain a lot of complex life cycles,
perform careful cleanup, and manage a considerable amount of state.

### 5.1.0 - Standard Interfaces for Telemetry Sources

Maintaining a strong separation of concerns between visualization components and data
sources is critical. To do this, we define various contracts for rendering different
types of telemetry sources.

```ts
// An interface for telemetry sources that provides a uniform set of values for an X
// and Y axis.
export interface XYTelemSource {
  x: () => Promise<Series[]>;
  y: () => Promise<Series[]>;
  xBounds: () => Promise<Bounds>;
  yBounds: () => Promise<Bounds>;
}

// A telemetry source for a single point value.
export interface NumericTelemSource {
  value: () => Promise<number>;
}

export interface CrudeColorTelemSource {
  value: () => Promise<Color>;
}

export interface BooleanTelemSource {
  value: () => boolean;
}
```

As an example, an Aether `Line` component can accept an `XYTelemSource` and then use a
WebGL rendering context to draw that line to the screen. A `Valve` component can use a
`BooleanTelemSource` to determine the valve state.

It's easy to see how we can compose and extend telemetry sources to alter their
functionality. For example, we could wrap several different numeric sources representing
data from different channels, execute some equation on them, and then expose the result
as another source with an identical interfaces. This pattern is remarkably powerful, and
allows us to provide fine-grained transformations on data to our users while adding
minimal complexity.

### 5.1.1 - Polymorphic Satellite Proxy

So how do we implement the above interfaces for accessing data from the database? The
best approach is to use a polymorphic satellite proxy. This is a fancy way of saying
that we maintain a client and caching mechanism in some central state, and then create
proxy objects that implement the above interfaces to call specific methods on the
client. This prevents us from overloading the client with implementing too much
functionality, and gives us close control over the lifecycle of telemetry arrays stored
in the cache.

## 5.2 - Scaling and Offsets

One of the key challenges I encountered with the previous visualization architecture was
developing a robust system for scaling, translating, and offsetting data for rendering.

1. Value in screen pixel space
2. Value in clip space
3. Value in data space

Storing a stateful value in decimal space should be used when an entity should remain in
the same position with respect to the plot viewport.

Storing a stateful value in screen pixel space should be used when an entity should
remain in the same position with respect to the screen.

Storing a stateful value in value space should be used when an entity should remain in
the same position with respect to the data.

Measure - store the value in decimal space.
