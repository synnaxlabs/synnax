// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Instrumentation, Logger, logThresholdFilter } from "@synnaxlabs/alamos";
import { RoutedWorker } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { flux } from "@/flux/aether";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { log } from "@/log/aether";
import { ontology } from "@/ontology/aether";
import { ranger } from "@/ranger/aether";
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
import { measure } from "@/vis/measure/aether";
import { rule } from "@/vis/rule/aether";
import { setpoint } from "@/vis/setpoint/aether";
import { toggle } from "@/vis/toggle/aether";
import { value } from "@/vis/value/aether";

const STORE_CONFIG: flux.StoreConfig<{
  [ranger.FLUX_STORE_KEY]: ranger.FluxStore;
  [ontology.RELATIONSHIPS_FLUX_STORE_KEY]: ontology.RelationshipFluxStore;
  [ontology.RESOURCES_FLUX_STORE_KEY]: ontology.ResourceFluxStore;
}> = {
  [ranger.FLUX_STORE_KEY]: ranger.FLUX_STORE_CONFIG,
  [ontology.RELATIONSHIPS_FLUX_STORE_KEY]: ontology.RELATIONSHIP_FLUX_STORE_CONFIG,
  [ontology.RESOURCES_FLUX_STORE_KEY]: ontology.RESOURCE_FLUX_STORE_CONFIG,
};

export const render = (): void => {
  // @ts-expect-error - for some reason post-message can't type transfer correctly
  const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
  onmessage = w.handle.bind(w);

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
    ...measure.REGISTRY,
    ...range.REGISTRY,
    ...rule.REGISTRY,
    ...setpoint.REGISTRY,
    ...status.REGISTRY,
    ...synnax.REGISTRY,
    ...telem.REGISTRY,
    ...theming.REGISTRY,
    ...toggle.REGISTRY,
    ...tooltip.REGISTRY,
    ...value.REGISTRY,
    ...log.REGISTRY,
    ...table.REGISTRY,
    ...gauge.REGISTRY,
    ...flux.createRegistry({ storeConfig: STORE_CONFIG }),
  };

  void aether.render({
    comms: w.route("vis"),
    registry: REGISTRY,
    instrumentation: new Instrumentation({
      logger: new Logger({ filters: [logThresholdFilter("info")] }),
    }),
  });
};
