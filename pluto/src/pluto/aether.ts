// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { synnax } from "@/synnax/aether";
import { Controller } from "@/telem/control/aether/control";
import { Provider } from "@/telem/provider/aether/provider";
import { Provider } from "@/theming/aether/provider";
import { AetherCanvas } from "@/vis/canvas/aether";
import { Line } from "@/vis/line/aether/line";
import { AetherLinePlot } from "@/vis/lineplot/aether";
import { Measure } from "@/vis/measure/aether/measure";
import { AetherPID } from "@/vis/pid/aether";
import { Rule } from "@/vis/rule/aether/aether";
import { Tooltip } from "@/vis/tooltip/aether/tooltip";
import { Value } from "@/vis/value/aether/value";
import { Valve } from "@/vis/valve/aether/valve";

export const pluto = (): void => {
  // @ts-expect-error
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

  const REGISTRY: aether.ComponentRegistry = {
    ...AetherLinePlot.REGISTRY,
    ...AetherCanvas.REGISTRY,
    [synnax.Provider.TYPE]: synnax.Provider,
    [Provider.TYPE]: Provider,
    [Line.TYPE]: Line,
    [Value.TYPE]: Value,
    [Valve.TYPE]: Valve,
    [AetherPID.TYPE]: AetherPID,
    [Rule.TYPE]: Rule,
    [Provider.TYPE]: Provider,
    [Tooltip.TYPE]: Tooltip,
    [Measure.TYPE]: Measure,
    [Controller.TYPE]: Controller,
  };

  aether.render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
