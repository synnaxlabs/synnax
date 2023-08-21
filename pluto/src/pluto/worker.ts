// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherComponentRegistry, render } from "@/aether/aether";
import { AetherClient } from "@/client/aether/provider";
import { AetherController } from "@/telem/control/aether";
import { AetherTelemProvider } from "@/telem/TelemProvider/aether";
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

  const REGISTRY: AetherComponentRegistry = {
    ...AetherLinePlot.REGISTRY,
    ...AetherCanvas.REGISTRY,
    [AetherClient.Provider.TYPE]: AetherClient.Provider,
    [AetherTelemProvider.TYPE]: AetherTelemProvider,
    [Line.TYPE]: Line,
    [Value.TYPE]: Value,
    [Valve.TYPE]: Valve,
    [AetherPID.TYPE]: AetherPID,
    [Rule.TYPE]: Rule,
    [Provider.TYPE]: Provider,
    [Tooltip.TYPE]: Tooltip,
    [Measure.TYPE]: Measure,
    [AetherController.TYPE]: AetherController,
  };

  render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
