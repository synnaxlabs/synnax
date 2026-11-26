// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import type { Names } from "@/schematic/node/builtIns";
import { SYMBOL as CAP } from "@/schematic/node/fittings/Cap";
import { SYMBOL as FILTER } from "@/schematic/node/fittings/Filter";
import { SYMBOL as FLOW_STRAIGHTENER } from "@/schematic/node/fittings/FlowStraightener";
import { SYMBOL as HEATER_ELEMENT } from "@/schematic/node/fittings/HeaterElement";
import { SYMBOL as ISO_CAP } from "@/schematic/node/fittings/ISOCap";
import { SYMBOL as ISO_FILTER } from "@/schematic/node/fittings/ISOFilter";
import { SYMBOL as NOZZLE } from "@/schematic/node/fittings/Nozzle";
import { SYMBOL as ORIFICE } from "@/schematic/node/fittings/Orifice";
import { SYMBOL as ORIFICE_PLATE } from "@/schematic/node/fittings/OrificePlate";
import { SYMBOL as STRAINER } from "@/schematic/node/fittings/Strainer";
import { SYMBOL as STRAINER_CONE } from "@/schematic/node/fittings/StrainerCone";
import { SYMBOL as THRUSTER } from "@/schematic/node/fittings/Thruster";
import { SYMBOL as VENT } from "@/schematic/node/fittings/Vent";

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

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  cap: { name: "Cap", label: "Cap" },
  filter: { name: "Filter", label: "Filter" },
  flow_straightener: { name: "Flow straightener", label: "Flow straightener" },
  heater_element: { name: "Heater", label: "Heater element" },
  iso_cap: { name: "ISO cap", label: "ISO cap" },
  iso_filter: { name: "ISO filter", label: "ISO filter" },
  nozzle: { name: "Nozzle", label: "Nozzle" },
  orifice: { name: "Orifice", label: "Orifice" },
  orifice_plate: { name: "Plate", label: "Orifice plate" },
  strainer: { name: "Strainer", label: "Strainer" },
  strainer_cone: { name: "Cone", label: "Strainer cone" },
  thruster: { name: "Thruster", label: "Thruster" },
  vent: { name: "Vent", label: "Vent" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
