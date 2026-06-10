// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createStatic, createToggle } from "@/schematic/node/common/create";
import { Cap } from "@/schematic/node/fittings/Cap";
import { Filter } from "@/schematic/node/fittings/Filter";
import { FlowStraightener } from "@/schematic/node/fittings/FlowStraightener";
import { HeaterElement } from "@/schematic/node/fittings/HeaterElement";
import { ISOCap } from "@/schematic/node/fittings/ISOCap";
import { ISOFilter } from "@/schematic/node/fittings/ISOFilter";
import { Nozzle } from "@/schematic/node/fittings/Nozzle";
import { Orifice } from "@/schematic/node/fittings/Orifice";
import { OrificePlate } from "@/schematic/node/fittings/OrificePlate";
import { Strainer } from "@/schematic/node/fittings/Strainer";
import { StrainerCone } from "@/schematic/node/fittings/StrainerCone";
import { Thruster } from "@/schematic/node/fittings/Thruster";
import { Vent } from "@/schematic/node/fittings/Vent";

const cap = createStatic({ variant: "cap", name: "Cap", Primitive: Cap });
const filter = createStatic({ variant: "filter", name: "Filter", Primitive: Filter });
const flowStraightener = createStatic({
  variant: "flowStraightener",
  name: "Flow Straightener",
  Primitive: FlowStraightener,
});
const heaterElement = createStatic({
  variant: "heaterElement",
  name: "Heater",
  label: "Heater Element",
  Primitive: HeaterElement,
});
const isoCap = createStatic({ variant: "isoCap", name: "ISO Cap", Primitive: ISOCap });
const isoFilter = createStatic({
  variant: "isoFilter",
  name: "ISO Filter",
  Primitive: ISOFilter,
});
const nozzle = createStatic({ variant: "nozzle", name: "Nozzle", Primitive: Nozzle });
const orifice = createStatic({
  variant: "orifice",
  name: "Orifice",
  Primitive: Orifice,
});
const orificePlate = createStatic({
  variant: "orificePlate",
  name: "Plate",
  label: "Orifice Plate",
  Primitive: OrificePlate,
});
const strainer = createStatic({
  variant: "strainer",
  name: "Strainer",
  Primitive: Strainer,
});
const strainerCone = createStatic({
  variant: "strainerCone",
  name: "Cone",
  label: "Strainer Cone",
  Primitive: StrainerCone,
});
const thruster = createToggle({
  variant: "thruster",
  name: "Thruster",
  Primitive: Thruster,
  node: "labeled",
});
const vent = createStatic({ variant: "vent", name: "Vent", Primitive: Vent });

export const REGISTRY = {
  cap: cap.spec,
  filter: filter.spec,
  flowStraightener: flowStraightener.spec,
  heaterElement: heaterElement.spec,
  isoCap: isoCap.spec,
  isoFilter: isoFilter.spec,
  nozzle: nozzle.spec,
  orifice: orifice.spec,
  orificePlate: orificePlate.spec,
  strainer: strainer.spec,
  strainerCone: strainerCone.spec,
  thruster: thruster.spec,
  vent: vent.spec,
} as const;
