// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherTooltip } from "./core/vis/Tooltip/aether";

import { AetherComponentRegistry, render } from "@/core/aether/worker";
import { AetherThemeProvider } from "@/core/theming/aether";
import { AetherCanvas } from "@/core/vis/Canvas/aether";
import { AetherLine } from "@/core/vis/Line/aether";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { AetherMeasure } from "@/core/vis/Measure/aether";
import { AetherPID } from "@/core/vis/PID/aether";
import { AetherRule } from "@/core/vis/Rule/aether";
import { AetherValue } from "@/core/vis/Value/aether";
import { AetherValve } from "@/core/vis/Valve/aether";
import { Telem } from "@/telem/TelemProvider/aether";

export const pluto = (): void => {
  // @ts-expect-error
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

  const REGISTRY: AetherComponentRegistry = {
    ...AetherLinePlot.REGISTRY,
    ...AetherCanvas.REGISTRY,
    [Telem.TYPE]: Telem,
    [AetherLine.TYPE]: AetherLine,
    [AetherValue.TYPE]: AetherValue,
    [AetherValve.TYPE]: AetherValve,
    [AetherPID.TYPE]: AetherPID,
    [AetherRule.TYPE]: AetherRule,
    [AetherThemeProvider.TYPE]: AetherThemeProvider,
    [AetherTooltip.TYPE]: AetherTooltip,
    [AetherMeasure.TYPE]: AetherMeasure,
  };

  render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
