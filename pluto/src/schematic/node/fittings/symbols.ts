// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { createStatic, createToggle } from "@/schematic/node/common/create";
import { Cap, SYMBOL as CAP } from "@/schematic/node/fittings/Cap";
import { Filter, SYMBOL as FILTER } from "@/schematic/node/fittings/Filter";
import {
  FlowStraightener,
  SYMBOL as FLOW_STRAIGHTENER,
} from "@/schematic/node/fittings/FlowStraightener";
import {
  HeaterElement,
  SYMBOL as HEATER_ELEMENT,
} from "@/schematic/node/fittings/HeaterElement";
import { ISOCap, SYMBOL as ISO_CAP } from "@/schematic/node/fittings/ISOCap";
import { ISOFilter, SYMBOL as ISO_FILTER } from "@/schematic/node/fittings/ISOFilter";
import { Nozzle, SYMBOL as NOZZLE } from "@/schematic/node/fittings/Nozzle";
import { Orifice, SYMBOL as ORIFICE } from "@/schematic/node/fittings/Orifice";
import {
  OrificePlate,
  SYMBOL as ORIFICE_PLATE,
} from "@/schematic/node/fittings/OrificePlate";
import { Strainer, SYMBOL as STRAINER } from "@/schematic/node/fittings/Strainer";
import {
  StrainerCone,
  SYMBOL as STRAINER_CONE,
} from "@/schematic/node/fittings/StrainerCone";
import { SYMBOL as THRUSTER, Thruster } from "@/schematic/node/fittings/Thruster";
import { SYMBOL as VENT, Vent } from "@/schematic/node/fittings/Vent";

const cap = createStatic({ variant: "cap", name: "Cap", Primitive: Cap });
const filter = createStatic({ variant: "filter", name: "Filter", Primitive: Filter });
const flowStraightener = createStatic({
  variant: "flow_straightener",
  name: "Flow straightener",
  Primitive: FlowStraightener,
});
const heaterElement = createStatic({
  variant: "heater_element",
  name: "Heater",
  label: "Heater element",
  Primitive: HeaterElement,
});
const isoCap = createStatic({ variant: "iso_cap", name: "ISO cap", Primitive: ISOCap });
const isoFilter = createStatic({
  variant: "iso_filter",
  name: "ISO filter",
  Primitive: ISOFilter,
});
const nozzle = createStatic({ variant: "nozzle", name: "Nozzle", Primitive: Nozzle });
const orifice = createStatic({
  variant: "orifice",
  name: "Orifice",
  Primitive: Orifice,
});
const orificePlate = createStatic({
  variant: "orifice_plate",
  name: "Plate",
  label: "Orifice plate",
  Primitive: OrificePlate,
});
const strainer = createStatic({
  variant: "strainer",
  name: "Strainer",
  Primitive: Strainer,
});
const strainerCone = createStatic({
  variant: "strainer_cone",
  name: "Cone",
  label: "Strainer cone",
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
  flow_straightener: flowStraightener.spec,
  heater_element: heaterElement.spec,
  iso_cap: isoCap.spec,
  iso_filter: isoFilter.spec,
  nozzle: nozzle.spec,
  orifice: orifice.spec,
  orifice_plate: orificePlate.spec,
  strainer: strainer.spec,
  strainer_cone: strainerCone.spec,
  thruster: thruster.spec,
  vent: vent.spec,
} as const;

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  cap: CAP,
  filter: FILTER,
  flow_straightener: FLOW_STRAIGHTENER,
  heater_element: HEATER_ELEMENT,
  iso_cap: ISO_CAP,
  iso_filter: ISO_FILTER,
  nozzle: NOZZLE,
  orifice: ORIFICE,
  orifice_plate: ORIFICE_PLATE,
  strainer: STRAINER,
  strainer_cone: STRAINER_CONE,
  thruster: THRUSTER,
  vent: VENT,
} as const satisfies Record<string, schematic.symbol.Spec>;
