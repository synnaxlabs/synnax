// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Instrumentation, Logger, logThresholdFilter } from "@synnaxlabs/alamos";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { color } from "@/color/aether";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { log, LogFactory } from "@/log/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { table } from "@/table/aether";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { theming } from "@/theming/aether";
import { button } from "@/vis/button/aether";
import { canvas } from "@/vis/canvas/aether";
import { diagram } from "@/vis/diagram/aether";
import { eraser } from "@/vis/eraser/aether";
import { gauge } from "@/vis/gauge/aether";
import { input } from "@/vis/input/aether";
import { latestSample } from "@/vis/latestSample/aether";
import { line } from "@/vis/line/aether";
import { scale } from "@/vis/scale/aether";
import { setpoint } from "@/vis/setpoint/aether";
import { staleness } from "@/vis/staleness/aether";
import { stateIndicator } from "@/vis/stateIndicator/aether";
import { stringValue } from "@/vis/stringValue/aether";
import { symbolNode } from "@/vis/symbolNode/aether";
import { value } from "@/vis/value/aether";

export const render = (): void => {
  const REGISTRY: aether.ComponentRegistry = {
    ...alamos.REGISTRY,
    ...button.REGISTRY,
    ...canvas.REGISTRY,
    ...control.REGISTRY,
    ...diagram.REGISTRY,
    ...eraser.REGISTRY,
    ...input.REGISTRY,
    ...latestSample.REGISTRY,
    ...line.REGISTRY,
    ...lineplot.REGISTRY,
    ...range.REGISTRY,
    ...scale.REGISTRY,
    ...color.REGISTRY,
    ...setpoint.REGISTRY,
    ...staleness.REGISTRY,
    ...stateIndicator.REGISTRY,
    ...status.REGISTRY,
    ...stringValue.REGISTRY,
    ...symbolNode.REGISTRY,
    ...synnax.REGISTRY,
    ...telem.createRegistry((client) => new LogFactory(client)),
    ...theming.REGISTRY,
    ...value.REGISTRY,
    ...log.REGISTRY,
    ...table.REGISTRY,
    ...gauge.REGISTRY,
  };

  aether.render({
    registry: REGISTRY,
    instrumentation: new Instrumentation({
      logger: new Logger({ filters: [logThresholdFilter("info")] }),
    }),
  });
};
