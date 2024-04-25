import { useState } from "react";

import { Button, Channel, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp, xy } from "@synnaxlabs/x";

export function App() {
  return (
    <>
      <Button.Button>My Cool Button</Button.Button>
      <Channel.LinePlot
        lines={[
          {
            key: "line1",
            axes: { x: "x", y: "y" },
            channels: {
              x: "stream_write_example_time",
              y: "stream_write_example_data_1",
            },
            variant: "dynamic",
            color: "#E87E04",
            label: "Example Data 1",
            strokeWidth: 3,
            // timeRange: new TimeRange({
            //   start: TimeStamp.now().sub(TimeSpan.minutes(1)),
            //   end: TimeStamp.now().add(TimeSpan.minutes(1)),
            // }),
            timeSpan: TimeSpan.seconds(30),
          },
        ]}
        axes={[
          {
            key: "x",
            label: "Time",
            location: "bottom",
            color: "#FFFFFF",
            type: "time",
          },
          {
            key: "y",
            label: "Value",
            location: "left",
            color: "#FFFFFF",
          },
        ]}
      />
    </>
  );
}

export default App;
