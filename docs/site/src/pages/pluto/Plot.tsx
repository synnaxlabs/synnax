import { useState } from "react";

import { Canvas, Channel, Pluto, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeSpan, xy } from "@synnaxlabs/x";

export function Plot() {
  return (
    <Pluto.Provider
      workerURL="/public/worker.js"
      connParams={{
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: false,
      }}
    >
      <PlotI />
    </Pluto.Provider>
  );
}

function PlotI() {
  const client = Synnax.use();
  const [lines, setLines] = useState<Channel.LineProps[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const [x, y] = await client.channels.retrieve([
      "stream_write_example_time",
      "stream_write_example_data_1",
    ]);
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
    <Canvas.Canvas
      style={{
        position: "absolute",
        width: 600,
        height: 600,
      }}
    >
      <Channel.LinePlot
        style={{
          width: 500,
          height: 500,
        }}
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
    </Canvas.Canvas>
  );
}
