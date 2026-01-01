// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxParams } from "@synnaxlabs/client";
import { Canvas, Channel, Pluto } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import WorkerURL from "@/components/pluto/worker?worker&url";

const connParams: SynnaxParams = {
  host: "demo.synnaxlabs.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
};

const providerProps: Pluto.ProviderProps = {
  theming: {
    applyCSSVars: false,
    theme: {
      key: "my-theme",
      colors: { primary: "#be9223" },
    },
  },
  connParams,
};

const AXES: Channel.AxisProps[] = [
  { key: "x", label: "Time", location: "bottom", type: "time" },
  { key: "y", label: "Value", location: "left" },
];

const LINES: Channel.BaseLineProps[] = [
  {
    key: "line1",
    axes: { x: "x", y: "y" },
    channels: { x: "stream_write_example_time", y: "stream_write_example_data_1" },
    color: "#3774d0",
    label: "Line 1",
    strokeWidth: 3,
  },
];

// eslint-disable-next-line react/display-name
export const factory = (props: Channel.LinePlotProps) => (): ReactElement => (
  <Pluto.Provider workerURL={WorkerURL} {...providerProps}>
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
      start: TimeStamp.now().sub(TimeSpan.seconds(30)),
      end: TimeStamp.now(),
    }),
  })),
});
