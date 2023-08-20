// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherClient } from "@/client/aether/provider";
import { AetherComponentRegistry, render } from "@/aether/aether";
import { Provider } from "@/theming/aether/provider";
import { AetherCanvas } from "@/core/vis/Canvas/aether";
import { AetherLine } from "@/core/vis/Line/aether";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { AetherMeasure } from "@/core/vis/Measure/aether";
import { AetherPID } from "@/core/vis/PID/aether";
import { AetherRule } from "@/core/vis/Rule/aether";
import { AetherTooltip } from "@/core/vis/Tooltip/aether";
import { AetherValue } from "@/core/vis/Value/aether";
import { AetherValve } from "@/core/vis/Valve/aether";
import { AetherController } from "@/telem/control/aether";
import { AetherTelemProvider } from "@/telem/TelemProvider/aether";

export const pluto = (): void => {
  // @ts-expect-error
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

  const REGISTRY: AetherComponentRegistry = {
    ...AetherLinePlot.REGISTRY,
    ...AetherCanvas.REGISTRY,
    [AetherClient.Provider.TYPE]: AetherClient.Provider,
    [AetherTelemProvider.TYPE]: AetherTelemProvider,
    [AetherLine.TYPE]: AetherLine,
    [AetherValue.TYPE]: AetherValue,
    [AetherValve.TYPE]: AetherValve,
    [AetherPID.TYPE]: AetherPID,
    [AetherRule.TYPE]: AetherRule,
    [Provider.TYPE]: Provider,
    [AetherTooltip.TYPE]: AetherTooltip,
    [AetherMeasure.TYPE]: AetherMeasure,
    [AetherController.TYPE]: AetherController,
  };

  render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
