// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { bounds, color } from "@synnaxlabs/x";

import { Label } from "@/schematic/node/common/label";
import { GaugeForm } from "@/schematic/node/general/gauge/Form";
import { Gauge } from "@/schematic/node/general/gauge/Primitive";
import { Symbol } from "@/schematic/node/general/gauge/Symbol";
import { type Spec } from "@/schematic/node/spec";

export const defaultConfig = (): schematic.NodeConfigGauge => ({
  variant: "gauge",
  orientation: "left",
  color: color.ZERO,
  units: "RPM",
  level: "h5",
  bounds: bounds.construct(0, 100),
  barWidth: 10,
  label: Label.defaultConfig("Gauge"),
});

export const spec: Spec<"gauge", schematic.NodeConfigGauge> = {
  key: "gauge",
  name: "Gauge",
  Form: GaugeForm,
  Node: Symbol,
  Preview: Gauge,
  defaultConfig,
  zIndex: 4,
  needsPosition: true,
};
