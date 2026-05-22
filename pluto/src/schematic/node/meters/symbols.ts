// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { defineStatic } from "@/schematic/node/define";
import { Primitive as Coriolis } from "@/schematic/node/meters/Coriolis";
import { Primitive as Electromagnetic } from "@/schematic/node/meters/Electromagnetic";
import { Primitive as FloatSensor } from "@/schematic/node/meters/FloatSensor";
import { Primitive as General } from "@/schematic/node/meters/General";
import { Primitive as Nozzle } from "@/schematic/node/meters/Nozzle";
import { Primitive as Orifice } from "@/schematic/node/meters/Orifice";
import { Primitive as PositiveDisplacement } from "@/schematic/node/meters/PositiveDisplacement";
import { Primitive as Pulse } from "@/schematic/node/meters/Pulse";
import { Primitive as RingPiston } from "@/schematic/node/meters/RingPiston";
import { Primitive as Turbine } from "@/schematic/node/meters/Turbine";
import { Primitive as VariableArea } from "@/schematic/node/meters/VariableArea";
import { Primitive as Venturi } from "@/schematic/node/meters/Venturi";

const general = defineStatic({
  variant: "flowmeterGeneral",
  name: "General",
  label: "General Flowmeter",
  Primitive: General,
});
const electromagnetic = defineStatic({
  variant: "flowmeterElectromagnetic",
  name: "Electromagnetic",
  label: "Electromagnetic Flowmeter",
  Primitive: Electromagnetic,
});
const variableArea = defineStatic({
  variant: "flowmeterVariableArea",
  name: "Variable Area",
  label: "Variable Area Flowmeter",
  Primitive: VariableArea,
});
const coriolis = defineStatic({
  variant: "flowmeterCoriolis",
  name: "Coriolis",
  label: "Coriolis Flowmeter",
  Primitive: Coriolis,
});
const nozzle = defineStatic({
  variant: "flowmeterNozzle",
  name: "Nozzle",
  label: "Nozzle Flowmeter",
  Primitive: Nozzle,
});
const venturi = defineStatic({
  variant: "flowmeterVenturi",
  name: "Venturi",
  label: "Venturi Flowmeter",
  Primitive: Venturi,
});
const ringPiston = defineStatic({
  variant: "flowmeterRingPiston",
  name: "Ring Piston",
  label: "Ring Piston Flowmeter",
  Primitive: RingPiston,
});
const positiveDisplacement = defineStatic({
  variant: "flowmeterPositiveDisplacement",
  name: "Positive Displacement",
  label: "Positive Displacement Flowmeter",
  Primitive: PositiveDisplacement,
});
const turbine = defineStatic({
  variant: "flowmeterTurbine",
  name: "Turbine",
  label: "Turbine Flowmeter",
  Primitive: Turbine,
});
const pulse = defineStatic({
  variant: "flowmeterPulse",
  name: "Pulse",
  label: "Pulse Flowmeter",
  Primitive: Pulse,
});
const floatSensor = defineStatic({
  variant: "flowmeterFloatSensor",
  name: "Float Sensor",
  label: "Float Sensor Flowmeter",
  Primitive: FloatSensor,
});
const orifice = defineStatic({
  variant: "flowmeterOrifice",
  name: "Orifice",
  label: "Orifice Flowmeter",
  Primitive: Orifice,
});

export const REGISTRY = {
  flowmeterGeneral: general.spec,
  flowmeterElectromagnetic: electromagnetic.spec,
  flowmeterVariableArea: variableArea.spec,
  flowmeterCoriolis: coriolis.spec,
  flowmeterNozzle: nozzle.spec,
  flowmeterVenturi: venturi.spec,
  flowmeterRingPiston: ringPiston.spec,
  flowmeterPositiveDisplacement: positiveDisplacement.spec,
  flowmeterTurbine: turbine.spec,
  flowmeterPulse: pulse.spec,
  flowmeterFloatSensor: floatSensor.spec,
  flowmeterOrifice: orifice.spec,
} as const;

export const configZ = z.discriminatedUnion("variant", [
  general.configZ,
  electromagnetic.configZ,
  variableArea.configZ,
  coriolis.configZ,
  nozzle.configZ,
  venturi.configZ,
  ringPiston.configZ,
  positiveDisplacement.configZ,
  turbine.configZ,
  pulse.configZ,
  floatSensor.configZ,
  orifice.configZ,
]);
