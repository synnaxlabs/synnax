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
import { flux } from "@/flux/aether";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { log } from "@/log/aether";
import { LogFactory } from "@/log/aether/telem/factory";
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
import { light } from "@/vis/light/aether";
import { line } from "@/vis/line/aether";
import { setpoint } from "@/vis/setpoint/aether";
import { stateIndicator } from "@/vis/stateIndicator/aether";
import { toggle } from "@/vis/toggle/aether";
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
    ...light.REGISTRY,
    ...line.REGISTRY,
    ...lineplot.REGISTRY,
    ...range.REGISTRY,
    ...setpoint.REGISTRY,
    ...stateIndicator.REGISTRY,
    ...status.REGISTRY,
    ...synnax.REGISTRY,
    ...telem.createRegistry((client) => new LogFactory(client)),
    ...theming.REGISTRY,
    ...toggle.REGISTRY,
    ...value.REGISTRY,
    ...log.REGISTRY,
    ...table.REGISTRY,
    ...gauge.REGISTRY,
    ...flux.createRegistry(),
  };

  void aether.render({
    registry: REGISTRY,
    instrumentation: new Instrumentation({
      logger: new Logger({ filters: [logThresholdFilter("info")] }),
    }),
  });
};
