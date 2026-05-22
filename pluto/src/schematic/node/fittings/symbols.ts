// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { defineStatic, defineToggle } from "@/schematic/node/define";
import { Cap } from "@/schematic/node/fittings/Cap";
import { Filter } from "@/schematic/node/fittings/Filter";
import { FlowStraightener } from "@/schematic/node/fittings/FlowStraightener";
import { HeaterElement } from "@/schematic/node/fittings/HeaterElement";
import { IsoCap } from "@/schematic/node/fittings/IsoCap";
import { IsoFilter } from "@/schematic/node/fittings/IsoFilter";
import { Nozzle } from "@/schematic/node/fittings/Nozzle";
import { Orifice } from "@/schematic/node/fittings/Orifice";
import { OrificePlate } from "@/schematic/node/fittings/OrificePlate";
import { Strainer } from "@/schematic/node/fittings/Strainer";
import { StrainerCone } from "@/schematic/node/fittings/StrainerCone";
import { Thruster } from "@/schematic/node/fittings/Thruster";
import { Vent } from "@/schematic/node/fittings/Vent";

const cap = defineStatic({
  variant: "cap",
  name: "Cap",
  label: "Cap",
  Primitive: Cap,
});
const filter = defineStatic({
  variant: "filter",
  name: "Filter",
  label: "Filter",
  Primitive: Filter,
});
const flowStraightener = defineStatic({
  variant: "flowStraightener",
  name: "Flow Straightener",
  label: "Flow Straightener",
  Primitive: FlowStraightener,
});
const heaterElement = defineStatic({
  variant: "heaterElement",
  name: "Heater",
  label: "Heater Element",
  Primitive: HeaterElement,
});
const isoCap = defineStatic({
  variant: "isoCap",
  name: "ISO Cap",
  label: "ISO Cap",
  Primitive: IsoCap,
});
const isoFilter = defineStatic({
  variant: "isoFilter",
  name: "ISO Filter",
  label: "ISO Filter",
  Primitive: IsoFilter,
});
const nozzle = defineStatic({
  variant: "nozzle",
  name: "Nozzle",
  label: "Nozzle",
  Primitive: Nozzle,
});
const orifice = defineStatic({
  variant: "orifice",
  name: "Orifice",
  label: "Orifice",
  Primitive: Orifice,
});
const orificePlate = defineStatic({
  variant: "orificePlate",
  name: "Plate",
  label: "Orifice Plate",
  Primitive: OrificePlate,
});
const strainer = defineStatic({
  variant: "strainer",
  name: "Strainer",
  label: "Strainer",
  Primitive: Strainer,
});
const strainerCone = defineStatic({
  variant: "strainerCone",
  name: "Cone",
  label: "Strainer Cone",
  Primitive: StrainerCone,
});
const thruster = defineToggle({
  variant: "thruster",
  name: "Thruster",
  label: "Thruster",
  Primitive: Thruster,
  node: "labeled",
});
const vent = defineStatic({
  variant: "vent",
  name: "Vent",
  label: "Vent",
  Primitive: Vent,
});

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

export const configZ = z.discriminatedUnion("variant", [
  cap.configZ,
  filter.configZ,
  flowStraightener.configZ,
  heaterElement.configZ,
  isoCap.configZ,
  isoFilter.configZ,
  nozzle.configZ,
  orifice.configZ,
  orificePlate.configZ,
  strainer.configZ,
  strainerCone.configZ,
  thruster.configZ,
  vent.configZ,
]);
