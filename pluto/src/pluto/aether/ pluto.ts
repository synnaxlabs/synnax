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
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { control } from "@/telem/control/aether";
import { provider } from "@/telem/provider/aether";
import { theming } from "@/theming/aether";
import { button } from "@/vis/button/aether";
import { canvas } from "@/vis/canvas/aether";
import { line } from "@/vis/line/aether";
import { lineplot } from "@/vis/lineplot/aether";
import { measure } from "@/vis/measure/aether";
import { pid } from "@/vis/pid/aether";
import { rule } from "@/vis/rule/aether";
import { tooltip } from "@/vis/tooltip/aether";
import { value } from "@/vis/value/aether";
import { valve } from "@/vis/valve/aether";

export const render = (): void => {
  // @ts-expect-error - for some reason post-message can't type transfer correctly
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

  const REGISTRY: aether.ComponentRegistry = {
    ...lineplot.REGISTRY,
    ...canvas.REGISTRY,
    ...synnax.REGISTRY,
    ...provider.REGISTRY,
    ...line.REGISTRY,
    ...value.REGISTRY,
    ...valve.REGISTRY,
    ...pid.REGISTRY,
    ...rule.REGISTRY,
    ...tooltip.REGISTRY,
    ...measure.REGISTRY,
    ...control.REGISTRY,
    ...theming.REGISTRY,
    ...status.REGISTRY,
    ...button.REGISTRY,
  };

  aether.render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
