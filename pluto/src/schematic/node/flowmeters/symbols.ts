// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { createStatic } from "@/schematic/node/common/create";
import { Coriolis, SYMBOL as CORIOLIS } from "@/schematic/node/flowmeters/Coriolis";
import {
  Electromagnetic,
  SYMBOL as ELECTROMAGNETIC,
} from "@/schematic/node/flowmeters/Electromagnetic";
import {
  FloatSensor,
  SYMBOL as FLOAT_SENSOR,
} from "@/schematic/node/flowmeters/FloatSensor";
import { General, SYMBOL as GENERAL } from "@/schematic/node/flowmeters/General";
import { Nozzle, SYMBOL as NOZZLE } from "@/schematic/node/flowmeters/Nozzle";
import { Orifice, SYMBOL as ORIFICE } from "@/schematic/node/flowmeters/Orifice";
import {
  PositiveDisplacement,
  SYMBOL as POSITIVE_DISPLACEMENT,
} from "@/schematic/node/flowmeters/PositiveDisplacement";
import { Pulse, SYMBOL as PULSE } from "@/schematic/node/flowmeters/Pulse";
import {
  RingPiston,
  SYMBOL as RING_PISTON,
} from "@/schematic/node/flowmeters/RingPiston";
import { SYMBOL as TURBINE, Turbine } from "@/schematic/node/flowmeters/Turbine";
import {
  SYMBOL as VARIABLE_AREA,
  VariableArea,
} from "@/schematic/node/flowmeters/VariableArea";
import { SYMBOL as VENTURI, Venturi } from "@/schematic/node/flowmeters/Venturi";

const general = createStatic({
  variant: "flowmeter_general",
  name: "General",
  label: "General flowmeter",
  Primitive: General,
});
const electromagnetic = createStatic({
  variant: "flowmeter_electromagnetic",
  name: "Electromagnetic",
  label: "Electromagnetic flowmeter",
  Primitive: Electromagnetic,
});
const variableArea = createStatic({
  variant: "flowmeter_variable_area",
  name: "Variable area",
  label: "Variable area flowmeter",
  Primitive: VariableArea,
});
const coriolis = createStatic({
  variant: "flowmeter_coriolis",
  name: "Coriolis",
  label: "Coriolis flowmeter",
  Primitive: Coriolis,
});
const nozzle = createStatic({
  variant: "flowmeter_nozzle",
  name: "Nozzle",
  label: "Nozzle flowmeter",
  Primitive: Nozzle,
});
const venturi = createStatic({
  variant: "flowmeter_venturi",
  name: "Venturi",
  label: "Venturi flowmeter",
  Primitive: Venturi,
});
const ringPiston = createStatic({
  variant: "flowmeter_ring_piston",
  name: "Ring piston",
  label: "Ring piston flowmeter",
  Primitive: RingPiston,
});
const positiveDisplacement = createStatic({
  variant: "flowmeter_positive_displacement",
  name: "Positive displacement",
  label: "Positive displacement flowmeter",
  Primitive: PositiveDisplacement,
});
const turbine = createStatic({
  variant: "flowmeter_turbine",
  name: "Turbine",
  label: "Turbine flowmeter",
  Primitive: Turbine,
});
const pulse = createStatic({
  variant: "flowmeter_pulse",
  name: "Pulse",
  label: "Pulse flowmeter",
  Primitive: Pulse,
});
const floatSensor = createStatic({
  variant: "flowmeter_float_sensor",
  name: "Float sensor",
  label: "Float sensor flowmeter",
  Primitive: FloatSensor,
});
const orifice = createStatic({
  variant: "flowmeter_orifice",
  name: "Orifice",
  label: "Orifice flowmeter",
  Primitive: Orifice,
});

export const REGISTRY = {
  flowmeter_general: general.spec,
  flowmeter_electromagnetic: electromagnetic.spec,
  flowmeter_variable_area: variableArea.spec,
  flowmeter_coriolis: coriolis.spec,
  flowmeter_nozzle: nozzle.spec,
  flowmeter_venturi: venturi.spec,
  flowmeter_ring_piston: ringPiston.spec,
  flowmeter_positive_displacement: positiveDisplacement.spec,
  flowmeter_turbine: turbine.spec,
  flowmeter_pulse: pulse.spec,
  flowmeter_float_sensor: floatSensor.spec,
  flowmeter_orifice: orifice.spec,
} as const;

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
