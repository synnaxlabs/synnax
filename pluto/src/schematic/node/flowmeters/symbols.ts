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
import { SYMBOL as CORIOLIS } from "@/schematic/node/flowmeters/Coriolis";
import { SYMBOL as ELECTROMAGNETIC } from "@/schematic/node/flowmeters/Electromagnetic";
import { SYMBOL as FLOAT_SENSOR } from "@/schematic/node/flowmeters/FloatSensor";
import { SYMBOL as GENERAL } from "@/schematic/node/flowmeters/General";
import { SYMBOL as NOZZLE } from "@/schematic/node/flowmeters/Nozzle";
import { SYMBOL as ORIFICE } from "@/schematic/node/flowmeters/Orifice";
import { SYMBOL as POSITIVE_DISPLACEMENT } from "@/schematic/node/flowmeters/PositiveDisplacement";
import { SYMBOL as PULSE } from "@/schematic/node/flowmeters/Pulse";
import { SYMBOL as RING_PISTON } from "@/schematic/node/flowmeters/RingPiston";
import { SYMBOL as TURBINE } from "@/schematic/node/flowmeters/Turbine";
import { SYMBOL as VARIABLE_AREA } from "@/schematic/node/flowmeters/VariableArea";
import { SYMBOL as VENTURI } from "@/schematic/node/flowmeters/Venturi";

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  flowmeter_coriolis: CORIOLIS,
  flowmeter_electromagnetic: ELECTROMAGNETIC,
  flowmeter_float_sensor: FLOAT_SENSOR,
  flowmeter_general: GENERAL,
  flowmeter_nozzle: NOZZLE,
  flowmeter_orifice: ORIFICE,
  flowmeter_positive_displacement: POSITIVE_DISPLACEMENT,
  flowmeter_pulse: PULSE,
  flowmeter_ring_piston: RING_PISTON,
  flowmeter_turbine: TURBINE,
  flowmeter_variable_area: VARIABLE_AREA,
  flowmeter_venturi: VENTURI,
} as const satisfies Record<string, schematic.symbol.Spec>;

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  flowmeter_coriolis: { name: "Coriolis", label: "Coriolis flowmeter" },
  flowmeter_electromagnetic: {
    name: "Electromagnetic",
    label: "Electromagnetic flowmeter",
  },
  flowmeter_float_sensor: { name: "Float sensor", label: "Float sensor flowmeter" },
  flowmeter_general: { name: "General", label: "General flowmeter" },
  flowmeter_nozzle: { name: "Nozzle", label: "Nozzle flowmeter" },
  flowmeter_orifice: { name: "Orifice", label: "Orifice flowmeter" },
  flowmeter_positive_displacement: {
    name: "Positive displacement",
    label: "Positive displacement flowmeter",
  },
  flowmeter_pulse: { name: "Pulse", label: "Pulse flowmeter" },
  flowmeter_ring_piston: { name: "Ring piston", label: "Ring piston flowmeter" },
  flowmeter_turbine: { name: "Turbine", label: "Turbine flowmeter" },
  flowmeter_variable_area: { name: "Variable area", label: "Variable area flowmeter" },
  flowmeter_venturi: { name: "Venturi", label: "Venturi flowmeter" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
