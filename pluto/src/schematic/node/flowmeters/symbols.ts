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
