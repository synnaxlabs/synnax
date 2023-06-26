# 13 - Pluto Visualization

**Feature Name**: Pluto Visualization <br />
**Start Date**: 2023-05-30 <br />
**Authors**: Emiliano Bonilla <br />
**Status**: Draft <br />

# 0 - Summary

Usable, high performance data visualization is at the core of what Synnax is offering.
The implementation of [telemetry streaming](./0012-230501-telemetry-streaming.md)
demands a significant change to how we approach the data fetching and rendering process.
The current design is also highly monolithic and tightly coupled. In this RFC I propose
a new architecture for Synnax's visualization system, implementing a modular
component based framework that shifts the responsibility of data fetching and rendering
completely off of the main thread. The goal is to keep the user facing API 'reacty'
while leveraging all the benefits of shifting the heavy lifting to a worker thread.

# 1 - Vocabulary

# 2 - Motivation

Line based visualization is by far the primary means of accessing the data that Synnax
stores. Many existing visualization systems feel clunky, and are typically intended
for static, small data sets. By providing an interface that allows access and
exploration of large, live data sets, we're empowering our users to take advantage of
all the advanced tooling Synnax has to offer, ultimately delivering our users a much
better understanding of how their systems are performing.

The current rendering pipeline requires reloading the entire data set on every update.
Now that we'll be updating at rates of 100Hz or more, this approach is no longer
sustainable. We need a method that only receives new data, and only renders the areas
of the canvas that have changed.

It also turns out that rendering lines and axes on a screen is remarkably complex,
requiring many coordinate transformations, GPU memory management, caching mechanisms,
and more. The current design decisions were made when we had far less knowledge of
the problem space and resulted in unorganized, tightly coupled code. We need a design
that provides clear isolation between different areas of the rendering process, allowing
us to make incremental improvements to each area without affecting others.

# 3 - Philosophy

## 3.0 - React Remains in Control

Pluto is a React component library, and maintaining a react-focused API lifts the burden
of implementing custom logic to decide the structure and lifecycle of visualization
components. A React-focused API is also more familiar to our users, allowing them to
use high-powered multithreaded visualizations as if they were working with simple
components.

## 3.1 - Offload the Heavy Lifting to a Worker Thread

The bulk of Pluto's memory usage and computation comes from fetching and rendering
buffers of telemetry. Common operations include counting minimums and maximums,
converting data types, and buffering values to the GPU. In many cases, these operations
can be so intense that they can block the main thread and prevent any user interactions.
As we add live telemetry, many of these processing operations aren't going to be user
driven, but will instead be triggered by incoming data from the server. By shifting
these expensive operations to a worker process, we can drastically reduce
the compute load on the main thread, leading to a much smoother experience for our
users even when working with large, real-time data sets.

## 3.2 - The Visualization Core Remains Generic

Pluto is a component library, and as such, the core visualization logic should remain
independent of any Synnax specific interfaces/telemetry systems. This provides us with
a clear separation of concerns that guarantees that as the Synnax Data API inevitably
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

On the worker thread, Aether maintains a composite tree of components whose implementation
feels similar to a Class-based React component. To fork a new component, we use the
`render` function, which takes in a registry of component factories. The `render` function
then receives messages from the main thread to update the component tree, creating and
destroying components as necessary.

```typescript
// worker.ts

import { Aether, AetherComponentRegistry } from '@synnaxlabs/pluto';
import { MyWorkerButton } from './MyWorkerButton';
import { MyWorkerLinePlot } from './MyWorkerLinePlot';

const REGISTRY: AetherComponentRegistry = {
    [MyWorkerButton.TYPE]: (initialState) => new MyWorkerButton(initialState),
    [MyWorkerLinePlot.TYPE]: (initialState) => new MyWorkerLinePlot(initialState),
};

Aether.render(REGISTRY)
```

There are two types of component that can be created in the Aether component tree:
`Composite` and `Leaf`. A `Composite` component can have children, while a `Leaf`
component cannot. To create a `Composite` component, we extend the `AetherComposite`
class, and, likewise, to create a `Leaf` component, we extend the `AetherLeaf` class:

```typescript
// MyWorkerButton.ts

import { AetherLeaf } from '@synnaxlabs/pluto';

export class MyWorkerButton extends AetherLeaf {
    static TYPE = 'MyWorkerButton';

    constructor(initialState) {
        super(initialState);
    }

    handleUpdate() {
        console.log("I'm doing something with state", this.state)
    }
}
```

It's important to note that the subclass we implement for an `AetherComposite` does
__not__ have control over the lifecycle of its children (can't create, delete, or set
the state). This is intentional, as the aether component tree is driven by React on
the main thread. The worker component tree does, however, have access to its children,
and can execute methods on them.

**Aether does not implement any rendering patterns**. All aether does is maintain a tree
of stateful components and allow the user to respond to state changes. In some cases,
a component may want to render something to the screen, but, in other cases, the
component may only be used for computation or data fetching.

### 5.0.1 - The Need for Context

Pluto makes extensive use of React's context API to provide components with access
to important tooling. The most notable example here is the visualization canvas, which
provides a WebGL rendering context to all components that need it.

## 5.0 - Visualization Component Structure

The most challenging roadblock with the previous visualization architecture was the
large, tightly coupled, and very complex functions and classes that handled the assembly
and drawing process. The separation of concerns was remarkably unclear, and refactoring
and adding features was remarkably challenging.

The new architecture separates these concerns by leveraging composition using React's
context API. The gist is to present a category of visualization as a container component
(i.e. `LinePlot`, `Valve`, or `Table`) and then allow the user to customize its layout
using children.

### 5.0.0 - The Line Plot Component

To demonstrate the flexibility of this approach, we'll use the line plot component
as an example. The code for a plot a single line is as follows:

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
additional axes, and more. This approach is extremely intuitive from a user perspective,
but it also gives us a clear method for separating concerns within the implementation.

## 5.1 - Integrating Telemetry Sources

## Defining Data Sources

## Dealing with Int64 Timestamps

Problem is less of a behavioral and more of a structural one.

Previous approach was to centralized. Instead of isolated areas of loosely
coupled, yet high complexity, we instead decided to distribute complexity and
create this sort of 'soup'.

We need to figure out how to break up and modularize this soup into concrete,
independently functioning pieces.

This is much more challenging given three realities:

- Web workers and offscreen canvas
- Caching
- Live telemetry

These three factors make visualization substantially harder.

The other is that there's not a clear unidirectional flow between
data source definition and render output. Sometimes the 'rendering'
logic such as an axis has an effect on the data source itself. When
we drag an axis or pan on a plot we load more data in.

We also want to keep data sources largely independent of anything Synnax
specific.

Another challenge is employing a centralized caching mechanism to reduce memory
usage.

- Polymorphic satellite proxy.

- Another thing to think about is data transformations
- Different visualizations have different requirements on their data source.
- Visualizations can define an interface for the data source they require.
- A single data source can implement multiple interfaces.

Data sources on the main thread are simply proxies to the actual
implementations on the worker thread.

So what would the data source for a line plot look like?

- Line plots require a time-aligned x and y-axis. So we need x-data
  and y-data so that the nth sample in each data 'array' lines up with
  each other time-wise, and (obviously) the two arrays are the same length.
- We also need a way to provide a bound with which to convert that data
  into screen space.

```jsx
<Axis.X>
    <Axis.Y>
        <DataSource/>
    </Axis.Y>
    <Axis.Y>
    </Axis.Y>
</Axis.X>
<DataSource/>
<Axis.X>
</Axis.X>
```

React components serve as proxies for worker render visualizations
and data structures.

It's the main threads job to define the 'structure' of the visualization.
It's the renderer threads job to do the data fetching and rendering. We want to be
able to define the structure in a semantic 'reacty' way. We also want it to be
intelligent
and opinionated.

We almost maintain a 'virtual DOM' on the worker side that mirrors the existing DOM,
but we let react do the hard part for us.

## 5.2 - Telemetry Infrastructure
