// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

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
  variant: "flowmeterGeneral",
  name: "General",
  label: "General flowmeter",
  Primitive: General,
});
const electromagnetic = createStatic({
  variant: "flowmeterElectromagnetic",
  name: "Electromagnetic",
  label: "Electromagnetic flowmeter",
  Primitive: Electromagnetic,
});
const variableArea = createStatic({
  variant: "flowmeterVariableArea",
  name: "Variable area",
  label: "Variable area flowmeter",
  Primitive: VariableArea,
});
const coriolis = createStatic({
  variant: "flowmeterCoriolis",
  name: "Coriolis",
  label: "Coriolis flowmeter",
  Primitive: Coriolis,
});
const nozzle = createStatic({
  variant: "flowmeterNozzle",
  name: "Nozzle",
  label: "Nozzle flowmeter",
  Primitive: Nozzle,
});
const venturi = createStatic({
  variant: "flowmeterVenturi",
  name: "Venturi",
  label: "Venturi flowmeter",
  Primitive: Venturi,
});
const ringPiston = createStatic({
  variant: "flowmeterRingPiston",
  name: "Ring piston",
  label: "Ring piston flowmeter",
  Primitive: RingPiston,
});
const positiveDisplacement = createStatic({
  variant: "flowmeterPositiveDisplacement",
  name: "Positive displacement",
  label: "Positive displacement flowmeter",
  Primitive: PositiveDisplacement,
});
const turbine = createStatic({
  variant: "flowmeterTurbine",
  name: "Turbine",
  label: "Turbine flowmeter",
  Primitive: Turbine,
});
const pulse = createStatic({
  variant: "flowmeterPulse",
  name: "Pulse",
  label: "Pulse flowmeter",
  Primitive: Pulse,
});
const floatSensor = createStatic({
  variant: "flowmeterFloatSensor",
  name: "Float sensor",
  label: "Float sensor flowmeter",
  Primitive: FloatSensor,
});
const orifice = createStatic({
  variant: "flowmeterOrifice",
  name: "Orifice",
  label: "Orifice flowmeter",
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
