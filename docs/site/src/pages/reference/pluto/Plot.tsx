import { useState } from "react";

import { Canvas, Channel, Pluto, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeSpan, xy } from "@synnaxlabs/x";

export function Plot() {
  return (
    <Pluto.Provider
      theming={{
        applyCSSVars: false,
      }}
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
  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: 500,
      }}
    >
      <Channel.LinePlot
        style={{
          width: "calc(100% - 3rem)",
          height: 500,
        }}
        clearOverScan={xy.ZERO}
        lines={[
          {
            key: "line1",
            axes: { x: "x", y: "y" },
            channels: {
              x: "stream_write_example_time",
              y: "stream_write_example_data_1",
            },
            variant: "dynamic",
            color: "#3774d0",
            label: "Line 1",
            strokeWidth: 3,
            timeSpan: TimeSpan.seconds(30),
          },
        ]}
        axes={[
          {
            key: "x",
            label: "Time",
            location: "bottom",
            type: "time",
          },
          {
            key: "y",
            label: "Value",
            location: "left",
          },
        ]}
      />
    </Canvas.Canvas>
  );
}
