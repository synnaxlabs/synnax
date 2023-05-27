## Defining Data Sources

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
    <DataSource />
    </Axis.Y>
    <Axis.Y>
    </Axis.Y>
    </Axis.X>
      <DataSource />
    <Axis.X>
</Axis.X>
```

React components serve as proxies for worker render visualizations
and data structures.

It's the main threads job to define the 'structure' of the visualization.
It's the renderer threads job to do the data fetching and rendering. We want to be
able to define the structure in a semantic 'reacty' way. We also want it to be intelligent
and opinionated.

We almost maintain a 'virtual DOM' on the worker side that mirrors the existing DOM,
but we let react do the hard part for us.
