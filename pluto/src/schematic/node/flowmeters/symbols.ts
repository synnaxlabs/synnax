// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createStatic } from "@/schematic/node/common/create";
import { Coriolis } from "@/schematic/node/flowmeters/Coriolis";
import { Electromagnetic } from "@/schematic/node/flowmeters/Electromagnetic";
import { FloatSensor } from "@/schematic/node/flowmeters/FloatSensor";
import { General } from "@/schematic/node/flowmeters/General";
import { Nozzle } from "@/schematic/node/flowmeters/Nozzle";
import { Orifice } from "@/schematic/node/flowmeters/Orifice";
import { PositiveDisplacement } from "@/schematic/node/flowmeters/PositiveDisplacement";
import { Pulse } from "@/schematic/node/flowmeters/Pulse";
import { RingPiston } from "@/schematic/node/flowmeters/RingPiston";
import { Turbine } from "@/schematic/node/flowmeters/Turbine";
import { VariableArea } from "@/schematic/node/flowmeters/VariableArea";
import { Venturi } from "@/schematic/node/flowmeters/Venturi";

const general = createStatic({
  variant: "flowmeter_general",
  name: "General",
  label: "General Flowmeter",
  Primitive: General,
});
const electromagnetic = createStatic({
  variant: "flowmeter_electromagnetic",
  name: "Electromagnetic",
  label: "Electromagnetic Flowmeter",
  Primitive: Electromagnetic,
});
const variableArea = createStatic({
  variant: "flowmeter_variable_area",
  name: "Variable Area",
  label: "Variable Area Flowmeter",
  Primitive: VariableArea,
});
const coriolis = createStatic({
  variant: "flowmeter_coriolis",
  name: "Coriolis",
  label: "Coriolis Flowmeter",
  Primitive: Coriolis,
});
const nozzle = createStatic({
  variant: "flowmeter_nozzle",
  name: "Nozzle",
  label: "Nozzle Flowmeter",
  Primitive: Nozzle,
});
const venturi = createStatic({
  variant: "flowmeter_venturi",
  name: "Venturi",
  label: "Venturi Flowmeter",
  Primitive: Venturi,
});
const ringPiston = createStatic({
  variant: "flowmeter_ring_piston",
  name: "Ring Piston",
  label: "Ring Piston Flowmeter",
  Primitive: RingPiston,
});
const positiveDisplacement = createStatic({
  variant: "flowmeter_positive_displacement",
  name: "Positive Displacement",
  label: "Positive Displacement Flowmeter",
  Primitive: PositiveDisplacement,
});
const turbine = createStatic({
  variant: "flowmeter_turbine",
  name: "Turbine",
  label: "Turbine Flowmeter",
  Primitive: Turbine,
});
const pulse = createStatic({
  variant: "flowmeter_pulse",
  name: "Pulse",
  label: "Pulse Flowmeter",
  Primitive: Pulse,
});
const floatSensor = createStatic({
  variant: "flowmeter_float_sensor",
  name: "Float Sensor",
  label: "Float Sensor Flowmeter",
  Primitive: FloatSensor,
});
const orifice = createStatic({
  variant: "flowmeter_orifice",
  name: "Orifice",
  label: "Orifice Flowmeter",
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
