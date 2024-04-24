/* eslint-disable react/display-name */
import { type ReactElement } from "react";

import { type SynnaxProps } from "@synnaxlabs/client";
import { Canvas, Channel, Pluto } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp, xy } from "@synnaxlabs/x";

const connParams: SynnaxProps = {
  host: "demo.synnaxlabs.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
};

const provideProps: Pluto.ProviderProps = {
  theming: { applyCSSVars: false, defaultTheme: "synnaxLight" },
  workerURL: "/public/worker.js",
  connParams,
};

const AXES: Channel.AxisProps[] = [
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
];

const LINES: Channel.BaseLineProps[] = [
  {
    key: "line1",
    axes: { x: "x", y: "y" },
    channels: {
      x: "stream_write_example_time",
      y: "stream_write_example_data_1",
    },
    color: "#3774d0",
    label: "Line 1",
    strokeWidth: 3,
  },
];

export const factory = (props: Channel.LinePlotProps) => (): ReactElement => (
  <Pluto.Provider {...provideProps}>
    <Canvas.Canvas style={{ width: "100%", height: 500 }}>
      <Channel.LinePlot
        style={{ width: "calc(100% - 3rem)", height: 500 }}
        clearOverScan={xy.ZERO}
        {...props}
      />
    </Canvas.Canvas>
  </Pluto.Provider>
);

export const RealTimePlot = factory({
  axes: AXES,
  lines: LINES.map((line) => ({
    ...line,
    variant: "dynamic",
    timeSpan: TimeSpan.seconds(30),
  })),
});

export const HistoricalPlot = factory({
  axes: AXES,
  lines: LINES.map((line) => ({
    ...line,
    variant: "static",
    timeRange: new TimeRange({
      start: TimeStamp.now().sub(TimeSpan.minutes(30)),
      end: TimeStamp.now(),
    }),
  })),
});
