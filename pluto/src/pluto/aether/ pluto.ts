// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RoutedWorker } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { chip } from "@/telem/control/chip/aether";
import { theming } from "@/theming/aether";
import { button } from "@/vis/button/aether";
import { canvas } from "@/vis/canvas/aether";
import { eraser } from "@/vis/eraser/aether";
import { line } from "@/vis/line/aether";
import { lineplot } from "@/vis/lineplot/aether";
import { tooltip } from "@/vis/lineplot/tooltip/aether";
import { measure } from "@/vis/measure/aether";
import { pid } from "@/vis/pid/aether";
import { rule } from "@/vis/rule/aether";
import { table } from "@/vis/table/aether";
import { toggle } from "@/vis/toggle/aether";
import { value } from "@/vis/value/aether";

export const render = (): void => {
  // @ts-expect-error - for some reason post-message can't type transfer correctly
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

  const REGISTRY: aether.ComponentRegistry = {
    ...lineplot.REGISTRY,
    ...canvas.REGISTRY,
    ...synnax.REGISTRY,
    ...telem.REGISTRY,
    ...line.REGISTRY,
    ...value.REGISTRY,
    ...toggle.REGISTRY,
    ...pid.REGISTRY,
    ...rule.REGISTRY,
    ...tooltip.REGISTRY,
    ...measure.REGISTRY,
    ...control.REGISTRY,
    ...theming.REGISTRY,
    ...status.REGISTRY,
    ...button.REGISTRY,
    ...chip.REGISTRY,
    ...alamos.REGISTRY,
    ...eraser.REGISTRY,
    ...table.REGISTRY,
  };

  aether.render({
    worker: w.route("vis"),
    registry: REGISTRY,
  });
};
