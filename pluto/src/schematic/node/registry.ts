// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import z from "zod";

import { Cap } from "@/schematic/node/fittings/cap";
import { Filter } from "@/schematic/node/fittings/filter";
import { FlowStraightener } from "@/schematic/node/fittings/flowStraightener";
import { HeaterElement } from "@/schematic/node/fittings/heaterElement";
import { IsoCap } from "@/schematic/node/fittings/isoCap";
import { IsoFilter } from "@/schematic/node/fittings/isoFilter";
import { Nozzle } from "@/schematic/node/fittings/nozzle";
import { Orifice } from "@/schematic/node/fittings/orifice";
import { OrificePlate } from "@/schematic/node/fittings/orificePlate";
import { Strainer } from "@/schematic/node/fittings/strainer";
import { StrainerCone } from "@/schematic/node/fittings/strainerCone";
import { Thruster } from "@/schematic/node/fittings/thruster";
import { Vent } from "@/schematic/node/fittings/vent";
import { Box } from "@/schematic/node/general/box";
import { Button } from "@/schematic/node/general/button";
import { Circle } from "@/schematic/node/general/circle";
import { CustomActuator } from "@/schematic/node/general/customActuator";
import { CustomStatic } from "@/schematic/node/general/customStatic";
import { Gauge } from "@/schematic/node/general/gauge";
import { Input } from "@/schematic/node/general/input";
import { Light } from "@/schematic/node/general/light";
import { OffPageReference } from "@/schematic/node/general/offPageReference";
import { Polygon } from "@/schematic/node/general/polygon";
import { Select } from "@/schematic/node/general/select";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { Switch } from "@/schematic/node/general/switch";
import { TextBox } from "@/schematic/node/general/textBox";
import { Value } from "@/schematic/node/general/value";
import { FlowmeterCoriolis } from "@/schematic/node/meters/coriolis";
import { FlowmeterElectromagnetic } from "@/schematic/node/meters/electromagnetic";
import { FlowmeterFloatSensor } from "@/schematic/node/meters/floatSensor";
import { FlowmeterGeneral } from "@/schematic/node/meters/general";
import { FlowmeterNozzle } from "@/schematic/node/meters/nozzle";
import { FlowmeterOrifice } from "@/schematic/node/meters/orifice";
import { FlowmeterPositiveDisplacement } from "@/schematic/node/meters/positiveDisplacement";
import { FlowmeterPulse } from "@/schematic/node/meters/pulse";
import { FlowmeterRingPiston } from "@/schematic/node/meters/ringPiston";
import { FlowmeterTurbine } from "@/schematic/node/meters/turbine";
import { FlowmeterVariableArea } from "@/schematic/node/meters/variableArea";
import { FlowmeterVenturi } from "@/schematic/node/meters/venturi";
import { Agitator } from "@/schematic/node/process/agitator";
import { CrossBeamAgitator } from "@/schematic/node/process/crossBeamAgitator";
import { FlatBladeAgitator } from "@/schematic/node/process/flatBladeAgitator";
import { HeatExchangerGeneral } from "@/schematic/node/process/heatExchangerGeneral";
import { HeatExchangerM } from "@/schematic/node/process/heatExchangerM";
import { HeatExchangerStraightTube } from "@/schematic/node/process/heatExchangerStraightTube";
import { HelicalAgitator } from "@/schematic/node/process/helicalAgitator";
import { PaddleAgitator } from "@/schematic/node/process/paddleAgitator";
import { PropellerAgitator } from "@/schematic/node/process/propellerAgitator";
import { RotaryMixer } from "@/schematic/node/process/rotaryMixer";
import { StaticMixer } from "@/schematic/node/process/staticMixer";
import { Cavity } from "@/schematic/node/pumps/cavity";
import { Centrifugal } from "@/schematic/node/pumps/centrifugal";
import { Compressor } from "@/schematic/node/pumps/compressor";
import { Diaphragm } from "@/schematic/node/pumps/diaphragm";
import { Ejection } from "@/schematic/node/pumps/ejection";
import { Ejector } from "@/schematic/node/pumps/ejector";
import { LiquidRing } from "@/schematic/node/pumps/liquidRing";
import { Piston } from "@/schematic/node/pumps/piston";
import { Pump } from "@/schematic/node/pumps/pump";
import { RollerVane } from "@/schematic/node/pumps/rollerVane";
import { Screw } from "@/schematic/node/pumps/screw";
import { Turbo } from "@/schematic/node/pumps/turbo";
import { Vacuum } from "@/schematic/node/pumps/vacuum";
import { BurstDisc } from "@/schematic/node/safety/burstDisc";
import { FlameArrestor } from "@/schematic/node/safety/flameArrestor";
import { FlameArrestorDetonation } from "@/schematic/node/safety/flameArrestorDetonation";
import { FlameArrestorExplosion } from "@/schematic/node/safety/flameArrestorExplosion";
import { FlameArrestorFireRes } from "@/schematic/node/safety/flameArrestorFireRes";
import { FlameArrestorFireResDetonation } from "@/schematic/node/safety/flameArrestorFireResDetonation";
import { IsoBurstDisc } from "@/schematic/node/safety/isoBurstDisc";
import { type Spec } from "@/schematic/node/spec";
import { Angled } from "@/schematic/node/valves/angled";
import { AngledRelief } from "@/schematic/node/valves/angledRelief";
import { AngledSpringLoadedRelief } from "@/schematic/node/valves/angledSpringLoadedRelief";
import { Ball } from "@/schematic/node/valves/ball";
import { Breather } from "@/schematic/node/valves/breather";
import { ButterflyOne } from "@/schematic/node/valves/butterflyOne";
import { ButterflyTwo } from "@/schematic/node/valves/butterflyTwo";
import { Check } from "@/schematic/node/valves/check";
import { CheckWithArrow } from "@/schematic/node/valves/checkWithArrow";
import { ElectricRegulator } from "@/schematic/node/valves/electricRegulator";
import { ElectricRegulatorMotorized } from "@/schematic/node/valves/electricRegulatorMotorized";
import { FourWay } from "@/schematic/node/valves/fourWay";
import { Gate } from "@/schematic/node/valves/gate";
import { IsoCheck } from "@/schematic/node/valves/isoCheck";
import { Manual } from "@/schematic/node/valves/manual";
import { Needle } from "@/schematic/node/valves/needle";
import { Regulator } from "@/schematic/node/valves/regulator";
import { RegulatorManual } from "@/schematic/node/valves/regulatorManual";
import { Relief } from "@/schematic/node/valves/relief";
import { Solenoid } from "@/schematic/node/valves/solenoid";
import { SpringLoadedRelief } from "@/schematic/node/valves/springLoadedRelief";
import { ThreeWay } from "@/schematic/node/valves/threeWay";
import { ThreeWayBall } from "@/schematic/node/valves/threeWayBall";
import { Valve } from "@/schematic/node/valves/valve";
import { CrossJunction } from "@/schematic/node/vessels/crossJunction";
import { Cylinder } from "@/schematic/node/vessels/cylinder";
import { Tank } from "@/schematic/node/vessels/tank";
import { TJunction } from "@/schematic/node/vessels/tJunction";

export const REGISTRY = {
  agitator: Agitator.spec,
  angledReliefValve: AngledRelief.spec,
  angledSpringLoadedReliefValve: AngledSpringLoadedRelief.spec,
  angledValve: Angled.spec,
  ballValve: Ball.spec,
  box: Box.spec,
  breatherValve: Breather.spec,
  burstDisc: BurstDisc.spec,
  butterflyValveOne: ButterflyOne.spec,
  butterflyValveTwo: ButterflyTwo.spec,
  button: Button.spec,
  cap: Cap.spec,
  cavityPump: Cavity.spec,
  centrifugalCompressor: Centrifugal.spec,
  checkValve: Check.spec,
  checkValveWithArrow: CheckWithArrow.spec,
  circle: Circle.spec,
  compressor: Compressor.spec,
  crossBeamAgitator: CrossBeamAgitator.spec,
  crossJunction: CrossJunction.spec,
  customActuator: CustomActuator.spec,
  customStatic: CustomStatic.spec,
  cylinder: Cylinder.spec,
  diaphragmPump: Diaphragm.spec,
  ejectionPump: Ejection.spec,
  ejectorCompressor: Ejector.spec,
  electricRegulator: ElectricRegulator.spec,
  electricRegulatorMotorized: ElectricRegulatorMotorized.spec,
  filter: Filter.spec,
  flameArrestor: FlameArrestor.spec,
  flameArrestorDetonation: FlameArrestorDetonation.spec,
  flameArrestorExplosion: FlameArrestorExplosion.spec,
  flameArrestorFireRes: FlameArrestorFireRes.spec,
  flameArrestorFireResDetonation: FlameArrestorFireResDetonation.spec,
  flatBladeAgitator: FlatBladeAgitator.spec,
  flowStraightener: FlowStraightener.spec,
  flowmeterCoriolis: FlowmeterCoriolis.spec,
  flowmeterElectromagnetic: FlowmeterElectromagnetic.spec,
  flowmeterFloatSensor: FlowmeterFloatSensor.spec,
  flowmeterGeneral: FlowmeterGeneral.spec,
  flowmeterNozzle: FlowmeterNozzle.spec,
  flowmeterOrifice: FlowmeterOrifice.spec,
  flowmeterPositiveDisplacement: FlowmeterPositiveDisplacement.spec,
  flowmeterPulse: FlowmeterPulse.spec,
  flowmeterRingPiston: FlowmeterRingPiston.spec,
  flowmeterTurbine: FlowmeterTurbine.spec,
  flowmeterVariableArea: FlowmeterVariableArea.spec,
  flowmeterVenturi: FlowmeterVenturi.spec,
  fourWayValve: FourWay.spec,
  gateValve: Gate.spec,
  gauge: Gauge.spec,
  heatExchangerGeneral: HeatExchangerGeneral.spec,
  heatExchangerM: HeatExchangerM.spec,
  heatExchangerStraightTube: HeatExchangerStraightTube.spec,
  heaterElement: HeaterElement.spec,
  helicalAgitator: HelicalAgitator.spec,
  input: Input.spec,
  isoBurstDisc: IsoBurstDisc.spec,
  isoCap: IsoCap.spec,
  isoCheckValve: IsoCheck.spec,
  isoFilter: IsoFilter.spec,
  light: Light.spec,
  liquidRingCompressor: LiquidRing.spec,
  manualValve: Manual.spec,
  needleValve: Needle.spec,
  nozzle: Nozzle.spec,
  offPageReference: OffPageReference.spec,
  orifice: Orifice.spec,
  orificePlate: OrificePlate.spec,
  paddleAgitator: PaddleAgitator.spec,
  pistonPump: Piston.spec,
  polygon: Polygon.spec,
  propellerAgitator: PropellerAgitator.spec,
  pump: Pump.spec,
  regulator: Regulator.spec,
  regulatorManual: RegulatorManual.spec,
  reliefValve: Relief.spec,
  rollerVaneCompressor: RollerVane.spec,
  rotaryMixer: RotaryMixer.spec,
  screwPump: Screw.spec,
  select: Select.spec,
  setpoint: Setpoint.spec,
  solenoidValve: Solenoid.spec,
  springLoadedReliefValve: SpringLoadedRelief.spec,
  stateIndicator: StateIndicator.spec,
  staticMixer: StaticMixer.spec,
  strainer: Strainer.spec,
  strainerCone: StrainerCone.spec,
  switch: Switch.spec,
  tJunction: TJunction.spec,
  tank: Tank.spec,
  textBox: TextBox.spec,
  threeWayBallValve: ThreeWayBall.spec,
  threeWayValve: ThreeWay.spec,
  thruster: Thruster.spec,
  turboCompressor: Turbo.spec,
  vacuumPump: Vacuum.spec,
  value: Value.spec,
  valve: Valve.spec,
  vent: Vent.spec,
} as const;

const VARIANTS = Object.keys(REGISTRY);
export const variantZ = z.enum(VARIANTS as [string, ...string[]]);
export type Variant = keyof typeof REGISTRY;

export const configZ = z.discriminatedUnion("variant", [
  Agitator.configZ,
  AngledRelief.configZ,
  AngledSpringLoadedRelief.configZ,
  Angled.configZ,
  Ball.configZ,
  Box.configZ,
  Breather.configZ,
  BurstDisc.configZ,
  ButterflyOne.configZ,
  ButterflyTwo.configZ,
  Button.configZ,
  Cap.configZ,
  Cavity.configZ,
  Centrifugal.configZ,
  Check.configZ,
  CheckWithArrow.configZ,
  Circle.configZ,
  Compressor.configZ,
  CrossBeamAgitator.configZ,
  CrossJunction.configZ,
  CustomActuator.configZ,
  CustomStatic.configZ,
  Cylinder.configZ,
  Diaphragm.configZ,
  Ejection.configZ,
  Ejector.configZ,
  ElectricRegulator.configZ,
  ElectricRegulatorMotorized.configZ,
  Filter.configZ,
  FlameArrestor.configZ,
  FlameArrestorDetonation.configZ,
  FlameArrestorExplosion.configZ,
  FlameArrestorFireRes.configZ,
  FlameArrestorFireResDetonation.configZ,
  FlatBladeAgitator.configZ,
  FlowStraightener.configZ,
  FlowmeterCoriolis.configZ,
  FlowmeterElectromagnetic.configZ,
  FlowmeterFloatSensor.configZ,
  FlowmeterGeneral.configZ,
  FlowmeterNozzle.configZ,
  FlowmeterOrifice.configZ,
  FlowmeterPositiveDisplacement.configZ,
  FlowmeterPulse.configZ,
  FlowmeterRingPiston.configZ,
  FlowmeterTurbine.configZ,
  FlowmeterVariableArea.configZ,
  FlowmeterVenturi.configZ,
  FourWay.configZ,
  Gate.configZ,
  Gauge.configZ,
  HeatExchangerGeneral.configZ,
  HeatExchangerM.configZ,
  HeatExchangerStraightTube.configZ,
  HeaterElement.configZ,
  HelicalAgitator.configZ,
  Input.configZ,
  IsoBurstDisc.configZ,
  IsoCap.configZ,
  IsoCheck.configZ,
  IsoFilter.configZ,
  Light.configZ,
  LiquidRing.configZ,
  Manual.configZ,
  Needle.configZ,
  Nozzle.configZ,
  OffPageReference.configZ,
  Orifice.configZ,
  OrificePlate.configZ,
  PaddleAgitator.configZ,
  Piston.configZ,
  Polygon.configZ,
  PropellerAgitator.configZ,
  Pump.configZ,
  Regulator.configZ,
  RegulatorManual.configZ,
  Relief.configZ,
  RollerVane.configZ,
  RotaryMixer.configZ,
  Screw.configZ,
  Select.configZ,
  Setpoint.configZ,
  Solenoid.configZ,
  SpringLoadedRelief.configZ,
  StateIndicator.configZ,
  StaticMixer.configZ,
  Strainer.configZ,
  StrainerCone.configZ,
  Switch.configZ,
  TJunction.configZ,
  Tank.configZ,
  TextBox.configZ,
  ThreeWayBall.configZ,
  ThreeWay.configZ,
  Thruster.configZ,
  Turbo.configZ,
  Vacuum.configZ,
  Value.configZ,
  Valve.configZ,
  Vent.configZ,
]);
export type Config = z.infer<typeof configZ>;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

export const resolveSpec = (variant: string): Spec<Variant, Config> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Symbol with variant ${variant} not found`);
  return spec as unknown as Spec<Variant, Config>;
};

/// CustomVariant is the union of Variants that reference a user-defined
/// symbol spec via specKey rather than rendering a hard-coded SVG.
export type CustomVariant = typeof CustomActuator.VARIANT | typeof CustomStatic.VARIANT;
export type CustomConfig = ConfigOf<CustomVariant>;

/// CUSTOM_VARIANTS is the set form of CustomVariant. Prefer the isCustomVariant
/// / isCustomConfig guards at call sites; expose the set for cases that need
/// to iterate the membership directly (e.g. tests).
export const CUSTOM_VARIANTS: ReadonlySet<Variant> = new Set<Variant>([
  CustomActuator.VARIANT,
  CustomStatic.VARIANT,
]);

export const isCustomVariant = (
  variant: string | undefined,
): variant is CustomVariant =>
  variant != null && CUSTOM_VARIANTS.has(variant as Variant);

export const isCustomConfig = (config: Config): config is CustomConfig =>
  isCustomVariant(config.variant);

/// STATIC_SPECS lists every Spec in the registry that is NOT a custom-symbol
/// variant. Used by the symbols toolbar to render the built-in catalog.
export const STATIC_SPECS: readonly Spec[] = (
  Object.values(REGISTRY) as ReadonlyArray<Spec>
).filter((s) => !isCustomVariant(s.key));
