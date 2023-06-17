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
a new architecture for Synnax's visualization system, that implements a modular
component based framework that shifts the responsibility of data fetching and rendering
completely off of the main thread. The goal is to keep the user facing API 'reacty'
while leveraging all the benefits of shifting the fetching and rendering process out of
React's control.

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

# Let React define the visualization structure and lifecycle

Pluto is a React component library, and maintaining a react-focused API lifts the burden
of implementing custom logic to decide the structure and lifecycle of visualization
components. A React-focused API is also more familiar to our users, allowing them to
use high-powered multithreaded visualizations as if they were working with simple
components.

# Move expensive fetching and rendering operations off of the main thread

The bulk of Pluto's memory usage and computation comes from fetching and rendering
buffers of telemetry. Common operations include counting minimums and maximums,
converting data types, and buffering values to the GPU. In many cases, these operations
can be so intense that they can block the main thread and prevent any user interactions.
As we add live telemetry, many of these processing operations aren't going to be user
driven, but will instead be triggered by incoming data from the server. By shifting
these expensive operations to a worker process, we can drastically reduce
the compute load on the main thread, leading to a much smoother experience for our
users even when working with large, real-time data sets.

# Maintain a service oriented architecture

# 5 - Detailed Design

## 5.0 - Line Plot Component Structure

The line plot component is designed as a tree of subcomponents that allows the user
to customize its structure. The idea here is that we let React and the DOM define both
the structure and the layout of the plot. We then mirror this DOM structure on the
worker thread, and, when our React components update, we send messages to the worker
thread to re-render. Ideally the worker thread would not send any messages back to
the main thread in order to reduce data transfer. The worker thread can also
independently re-render the plot on receiving updates from arbitrary telemetry sources.

The structure of a simple line plot would resemble the following:

```typescript jsx
<LinePlot>
    <LinePlot.Title>My Line Plot</LinePlot.Title>
    <LinePlot.XAxis name="x1" label="Time" location="bottom">
        <LinePlot.YAxis name="y1" label="Value" location="left">
            <LinePlot.Line name="line1" telem={someTelemSource}/>
        </LinePlot.Line>
    </LinePlot.XAxis>
</LinePlot>
```

## 5.1 - Defining the Worker Component Tree - Composite

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
