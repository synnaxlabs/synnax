// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxParams } from "@synnaxlabs/client";
import { Canvas, LinePlot, Pluto, telem } from "@synnaxlabs/pluto";
import { color, TimeRange, TimeSpan, TimeStamp, xy } from "@synnaxlabs/x";
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

const X_CHANNEL = "stream_write_example_time";
const Y_CHANNEL = "stream_write_example_data_1";

type Source = ReturnType<typeof telem.channelData>;

// eslint-disable-next-line react/display-name
const factory = (x: Source, y: Source) => (): ReactElement => (
  <Pluto.Provider workerURL={WorkerURL} {...providerProps}>
    <Canvas.Canvas style={{ width: "100%", height: 500 }}>
      <LinePlot.Frame
        style={{ width: "calc(100% - 3rem)", height: 500 }}
        clearOverScan={xy.ZERO}
      >
        <LinePlot.XAxis axisKey="x1" location="bottom" label="Time" type="time">
          <LinePlot.YAxis axisKey="y1" location="left" label="Value">
            <LinePlot.Line
              x={x}
              y={y}
              color={color.construct("#3774d0")}
              label="Line 1"
              strokeWidth={3}
              legendGroup="Value"
            />
          </LinePlot.YAxis>
        </LinePlot.XAxis>
      </LinePlot.Frame>
    </Canvas.Canvas>
  </Pluto.Provider>
);

export const RealTimePlot = factory(
  telem.streamChannelData({ timeSpan: TimeSpan.seconds(30), channel: X_CHANNEL }),
  telem.streamChannelData({ timeSpan: TimeSpan.seconds(30), channel: Y_CHANNEL }),
);

const historicalRange = new TimeRange({
  start: TimeStamp.now().sub(TimeSpan.seconds(30)),
  end: TimeStamp.now(),
});

export const HistoricalPlot = factory(
  telem.channelData({ timeRange: historicalRange, channel: X_CHANNEL }),
  telem.channelData({ timeRange: historicalRange, channel: Y_CHANNEL }),
);
