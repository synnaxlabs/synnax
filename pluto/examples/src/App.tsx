import { useState } from "react";

import { Channel, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeSpan, xy } from "@synnaxlabs/x";

export function Plot() {
  const client = Synnax.use();
  const [lines, setLines] = useState<Channel.LineProps[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const [x, y] = await client.channels.retrieve([
      "stream_write_example_time",
      "stream_write_example_data_1",
    ]);
    console.log(x, y);
    setLines([
      {
        key: "line1",
        axes: { x: "x", y: "y" },
        channels: { x: x.key, y: y.key },
        variant: "dynamic",
        color: "#E87E04",
        label: "Dog",
        strokeWidth: 2,
        timeSpan: TimeSpan.seconds(30),
      },
    ]);
  }, [client]);
  return (
    <Channel.LinePlot
      clearOverscan={xy.ZERO}
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
      lines={lines}
    />
  );
}

export default App;
