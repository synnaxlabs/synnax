// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/Symbols.css";

import { box, direction, location, type UnknownRecord, xy } from "@synnaxlabs/x";
import { type CSSProperties, type FC, type ReactElement } from "react";

import { Align } from "@/align";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { Control } from "@/telem/control";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Button as CoreButton } from "@/vis/button";
import { Light as CoreLight } from "@/vis/light";
import { Grid, type GridItem } from "@/vis/schematic/Grid";
import { Primitives } from "@/vis/schematic/primitives";
import { Setpoint as CoreSetpoint } from "@/vis/setpoint";
import { Toggle } from "@/vis/toggle";
import { Value as CoreValue } from "@/vis/value";

export interface ControlStateProps extends Omit<Align.SpaceProps, "direction"> {
  show?: boolean;
  showChip?: boolean;
  showIndicator?: boolean;
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
  orientation?: location.Outer;
}

export interface LabelExtensionProps {
  label?: string;
  level?: Text.Level;
  orientation?: location.Outer;
  direction?: direction.Direction;
  maxInlineSize?: number;
  align?: Align.Alignment;
}

const labelGridItem = (
  props?: LabelExtensionProps,
  onChange?: ({ label }: { label: LabelExtensionProps }) => void,
): GridItem | null => {
  if (props == null) return null;
  const {
    label,
    level = "p",
    orientation = "top",
    direction,
    align,
    maxInlineSize,
  } = props;
  if (label == null || label.length === 0) return null;
  return {
    key: "label",
    element: (
      <Text.Editable
        className={CSS(CSS.BE("symbol", "label"), CSS.dir(direction))}
        level={level}
        value={label}
        onChange={(value: string) => onChange?.({ label: { ...props, label: value } })}
        allowEmpty
        style={{ textAlign: align as CSSProperties["textAlign"], maxInlineSize }}
      />
    ),
    location: orientation,
  };
};

export type SymbolProps<P extends object = UnknownRecord> = P & {
  symbolKey: string;
  position: xy.XY;
  aetherKey: string;
  selected: boolean;
  draggable: boolean;
  onChange: (value: Partial<P>) => void;
};

const controlStateGridItem = (props?: ControlStateProps): GridItem | null => {
  if (props == null) return null;
  const {
    show = true,
    showChip = true,
    showIndicator = true,
    chip,
    indicator,
    orientation = "bottom",
  } = props;
  return {
    key: "control",
    element: (
      <Align.Space
        direction={direction.swap(orientation)}
        align="center"
        className={CSS(CSS.B("control-state"))}
        size="small"
      >
        {show && showChip && <Control.Chip size="small" {...chip} />}
        {show && showIndicator && <Control.Indicator {...indicator} />}
      </Align.Space>
    ),
    location: orientation,
  };
};

export type ToggleProps<T> = T &
  Omit<Toggle.UseProps, "aetherKey" | "onChange"> & {
    label?: LabelExtensionProps;
    control?: ControlStateProps;
    orientation?: location.Outer;
  };

export const createToggle = <P extends object = UnknownRecord>(BaseSymbol: FC<P>) => {
  const C = ({
    symbolKey,
    control,
    source,
    sink,
    label,
    onChange,
    draggable,
    selected,
    orientation = "left",
    ...rest
  }: SymbolProps<ToggleProps<P>>): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({
      aetherKey: symbolKey,
      source,
      sink,
    });
    const gridItems: GridItem[] = [];
    /* @ts-expect-error - typescript with HOCs */
    const labelItem = labelGridItem(label, onChange);
    if (labelItem != null) gridItems.push(labelItem);
    const controlItem = controlStateGridItem(control);
    if (controlItem != null) gridItems.push(controlItem);
    return (
      <Grid
        editable={selected && !draggable}
        symbolKey={symbolKey}
        items={gridItems}
        onRotate={() =>
          onChange({ orientation: location.rotate90(orientation) } as Partial<
            ToggleProps<P>
          >)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({ label: { ...label, orientation: loc } } as Partial<
              ToggleProps<P>
            >);
          if (key === "control")
            onChange({ control: { ...control, orientation: loc } } as Partial<
              ToggleProps<P>
            >);
        }}
      >
        {/* @ts-expect-error - typescript with HOCs */}
        <BaseSymbol
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          orientation={orientation}
          {...rest}
        />
      </Grid>
    );
  };
  C.displayName = BaseSymbol.displayName;
  return C;
};

type LabeledProps<P extends object = UnknownRecord> = P & {
  label?: LabelExtensionProps;
  orientation?: location.Outer;
};

export const createLabeled = <P extends object = UnknownRecord>(BaseSymbol: FC<P>) => {
  const C = ({
    symbolKey,
    label,
    onChange,
    selected,
    draggable,
    position: _,
    orientation = "left",
    ...rest
  }: SymbolProps<LabeledProps<P>>): ReactElement => {
    const gridItems: GridItem[] = [];
    /* @ts-expect-error - typescript with HOCs */
    const labelItem = labelGridItem(label, onChange);
    if (labelItem != null) gridItems.push(labelItem);
    return (
      <Grid
        items={gridItems}
        editable={selected && !draggable}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({ orientation: location.rotate90(orientation) } as Partial<
            LabeledProps<P>
          >)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<LabeledProps<P>>);
        }}
      >
        {/* @ts-expect-error - typescript with HOCs */}
        <BaseSymbol orientation={orientation} {...rest} />
      </Grid>
    );
  };
  C.displayName = BaseSymbol.displayName;
  return C;
};

type DummyToggleProps<P extends object = UnknownRecord> = LabeledProps<P> & {
  enabled?: boolean;
  clickable?: boolean;
};

export const createDummyToggle = <P extends object = UnknownRecord>(
  Primitive: FC<P>,
) => {
  const DummyToggle = ({
    symbolKey,
    label,
    onChange,
    selected,
    draggable,
    position: _,
    orientation = "left",
    enabled = false,
    clickable = false,
    ...rest
  }: SymbolProps<DummyToggleProps<P>>): ReactElement => {
    const gridItems: GridItem[] = [];
    /* @ts-expect-error - typescript with HOCs */
    const labelItem = labelGridItem(label, onChange);
    if (labelItem != null) gridItems.push(labelItem);
    const handleToggleChange = () => {
      if (!clickable) return;
      onChange({ enabled: !enabled } as Partial<DummyToggleProps<P>>);
    };
    return (
      <Grid
        items={gridItems}
        editable={selected && !draggable}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({ orientation: location.rotate90(orientation) } as Partial<
            LabeledProps<P>
          >)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<LabeledProps<P>>);
        }}
      >
        {/* @ts-expect-error - typescript with HOCs */}
        <Primitive
          orientation={orientation}
          enabled={enabled}
          onClick={handleToggleChange}
          {...rest}
        />
      </Grid>
    );
  };
  DummyToggle.displayName = Primitive.displayName;
  return DummyToggle;
};

// ||||||||| TOGGLE ||||||||

export const ThreeWayValve = createToggle(Primitives.ThreeWayValve);
export type ThreeWayValveProps = ToggleProps<Primitives.ThreeWayValveProps>;
export const Valve = createToggle(Primitives.Valve);
export type ValveProps = ToggleProps<Primitives.ValveProps>;
export const SolenoidValve = createToggle(Primitives.SolenoidValve);
export type SolenoidValveProps = ToggleProps<Primitives.SolenoidValveProps>;
export const FourWayValve = createToggle(Primitives.FourWayValve);
export type FourWayValveProps = ToggleProps<Primitives.FourWayValveProps>;
export const AngledValve = createToggle(Primitives.AngledValve);
export type AngledValveProps = ToggleProps<Primitives.AngledValveProps>;
export const Pump = createToggle(Primitives.Pump);
export type PumpProps = ToggleProps<Primitives.PumpProps>;
export const DiaphragmPump = createToggle(Primitives.DiaphragmPump);
export type DiaphragmPumpProps = ToggleProps<Primitives.DiaphragmPumpProps>;
export const EjectionPump = createToggle(Primitives.EjectionPump);
export type EjectionPumpProps = ToggleProps<Primitives.EjectionPumpProps>;
export const VacuumPump = createToggle(Primitives.VacuumPump);
export type VacuumPumpProps = ToggleProps<Primitives.VacuumPumpProps>;
export const CavityPump = createToggle(Primitives.CavityPump);
export type CavityPumpProps = ToggleProps<Primitives.CavityPumpProps>;
export const PistonPump = createToggle(Primitives.PistonPump);
export type PistonPumpProps = ToggleProps<Primitives.PistonPumpProps>;
export const RotaryMixer = createToggle(Primitives.RotaryMixer);
export type RotaryMixerProps = ToggleProps<Primitives.RotaryMixerProps>;
export const Agitator = createToggle(Primitives.Agitator);
export type AgitatorProps = ToggleProps<Primitives.AgitatorProps>;
export const PropellerAgitator = createToggle(Primitives.PropellerAgitator);
export type PropellerAgitatorProps = ToggleProps<Primitives.PropellerAgitatorProps>;
export const FlatBladeAgitator = createToggle(Primitives.FlatBladeAgitator);
export type FlatBladeAgitatorProps = ToggleProps<Primitives.FlatBladeAgitatorProps>;
export const PaddleAgitator = createToggle(Primitives.PaddleAgitator);
export type PaddleAgitatorProps = ToggleProps<Primitives.PaddleAgitatorProps>;
export const CrossBeamAgitator = createToggle(Primitives.CrossBeamAgitator);
export type CrossBeamAgitatorProps = ToggleProps<Primitives.CrossBeamAgitatorProps>;
export const HelicalAgitator = createToggle(Primitives.HelicalAgitator);
export type HelicalAgitatorProps = ToggleProps<Primitives.HelicalAgitatorProps>;
export const ScrewPump = createToggle(Primitives.ScrewPump);
export type ScrewPumpProps = ToggleProps<Primitives.ScrewPumpProps>;
export const Compressor = createToggle(Primitives.Compressor);
export type CompressorProps = ToggleProps<Primitives.CompressorProps>;
export const TurboCompressor = createToggle(Primitives.TurboCompressor);
export type TurboCompressorProps = ToggleProps<Primitives.TurboCompressorProps>;
export const RollerVaneCompressor = createToggle(Primitives.RollerVaneCompressor);
export type RollerVaneCompressorProps =
  ToggleProps<Primitives.RollerVaneCompressorProps>;
export const LiquidRingCompressor = createToggle(Primitives.LiquidRingCompressor);
export type LiquidRingCompressorProps =
  ToggleProps<Primitives.LiquidRingCompressorProps>;
export const EjectorCompressor = createToggle(Primitives.EjectorCompressor);
export type EjectorCompressorProps = ToggleProps<Primitives.EjectorCompressorProps>;
export const CentrifugalCompressor = createToggle(Primitives.CentrifugalCompressor);
export type CentrifugalCompressorProps =
  ToggleProps<Primitives.CentrifugalCompressorProps>;
export const ButterflyValveOne = createToggle(Primitives.ButterflyValveOne);
export type ButterflyValveOneProps = ToggleProps<Primitives.ButterflyValveOneProps>;
export const ButterflyValveTwo = createToggle(Primitives.ButterflyValveTwo);
export type ButterflyValveTwoProps = ToggleProps<Primitives.ButterflyValveTwoProps>;
export const BallValve = createToggle(Primitives.BallValve);
export type BallValveProps = ToggleProps<Primitives.BallValveProps>;
export const ThreeWayBallValve = createToggle(Primitives.ThreeWayBallValve);
export type ThreeWayBallValveProps = ToggleProps<Primitives.ThreeWayBallValveProps>;
export const GateValve = createToggle(Primitives.GateValve);
export type GateValveProps = ToggleProps<Primitives.GateValveProps>;

// |||||||| STATIC + LABELED ||||||||

export const Regulator = createLabeled(Primitives.Regulator);
export type RegulatorProps = LabeledProps<Primitives.RegulatorProps>;
export const RegulatorManual = createLabeled(Primitives.RegulatorManual);
export type RegulatorManualProps = LabeledProps<Primitives.RegulatorManualProps>;
export const ElectricRegulator = createLabeled(Primitives.ElectricRegulator);
export type ElectricRegulatorProps = LabeledProps<Primitives.ElectricRegulatorProps>;
export const ElectricRegulatorMotorized = createLabeled(
  Primitives.ElectricRegulatorMotorized,
);
export type ElectricRegulatorMotorizedProps =
  LabeledProps<Primitives.ElectricRegulatorMotorizedProps>;
export const BurstDisc = createLabeled(Primitives.BurstDisc);
export type BurstDiscProps = LabeledProps<Primitives.BurstDiscProps>;
export const Cap = createLabeled(Primitives.Cap);
export type CapProps = LabeledProps<Primitives.CapProps>;
export const ISOCap = createLabeled(Primitives.ISOCap);
export type ISOCapProps = LabeledProps<Primitives.ISOCapProps>;
export const Filter = createLabeled(Primitives.Filter);
export type FilterProps = LabeledProps<Primitives.FilterProps>;
export const CheckValve = createLabeled(Primitives.CheckValve);
export type CheckValveProps = LabeledProps<Primitives.CheckValveProps>;
export const ISOCheckValve = createLabeled(Primitives.ISOCheckValve);
export type ISOCheckValveProps = LabeledProps<Primitives.ISOCheckValveProps>;
export const CheckValveWithArrow = createLabeled(Primitives.CheckValveWithArrow);
export type CheckValveWithArrowProps =
  LabeledProps<Primitives.CheckValveWithArrowProps>;
export const Orifice = createLabeled(Primitives.Orifice);
export type OrificeProps = LabeledProps<Primitives.OrificeProps>;
export const Switch = createToggle(Primitives.Switch);
export type SwitchProps = ToggleProps<Primitives.SwitchProps>;
export const Vent = createLabeled(Primitives.Vent);
export type VentProps = LabeledProps<Primitives.VentProps>;
export const OrificePlate = createLabeled(Primitives.OrificePlate);
export type OrificePlateProps = LabeledProps<Primitives.OrificePlateProps>;
export const ISOFilter = createLabeled(Primitives.ISOFilter);
export type ISOFilterProps = LabeledProps<Primitives.ISOFilterProps>;
export const FlowStraightener = createLabeled(Primitives.FlowStraightener);
export type FlowStraightenerProps = LabeledProps<Primitives.FlowStraightenerProps>;
export const HeaterElement = createLabeled(Primitives.HeaterElement);
export type HeaterElementProps = LabeledProps<Primitives.HeaterElementProps>;
export const ISOBurstDisc = createLabeled(Primitives.ISOBurstDisc);
export type ISOBurstDiscProps = LabeledProps<Primitives.ISOBurstDiscProps>;
export const TJunction = createLabeled(Primitives.TJunction);
export type TJunctionProps = LabeledProps<Primitives.TJunctionProps>;
export const CrossJunction = createLabeled(Primitives.CrossJunction);
export type CrossJunctionProps = LabeledProps<Primitives.CrossJunctionProps>;
export const StaticMixer = createLabeled(Primitives.StaticMixer);
export type StaticMixerProps = LabeledProps<Primitives.StaticMixerProps>;
export const FlowmeterGeneral = createLabeled(Primitives.FlowmeterGeneral);
export type FlowmeterGeneralProps = LabeledProps<Primitives.FlowmeterGeneralProps>;
export const FlowmeterElectromagnetic = createLabeled(
  Primitives.FlowmeterElectromagnetic,
);
export type FlowmeterElectromagneticProps =
  LabeledProps<Primitives.FlowmeterElectromagneticProps>;
export const FlowmeterVariableArea = createLabeled(Primitives.FlowmeterVariableArea);
export type FlowmeterVariableAreaProps =
  LabeledProps<Primitives.FlowmeterVariableAreaProps>;
export const FlowmeterCoriolis = createLabeled(Primitives.FlowmeterCoriolis);
export type FlowmeterCoriolisProps = LabeledProps<Primitives.FlowmeterCoriolisProps>;
export const FlowmeterNozzle = createLabeled(Primitives.FlowmeterNozzle);
export type FlowmeterNozzleProps = LabeledProps<Primitives.FlowmeterNozzleProps>;
export const FlowmeterVenturi = createLabeled(Primitives.FlowmeterVenturi);
export type FlowmeterVenturiProps = LabeledProps<Primitives.FlowmeterVenturiProps>;
export const FlowmeterRingPiston = createLabeled(Primitives.FlowmeterRingPiston);
export type FlowmeterRingPistonProps =
  LabeledProps<Primitives.FlowmeterRingPistonProps>;
export const FlowmeterPositiveDisplacement = createLabeled(
  Primitives.FlowmeterPositiveDisplacement,
);
export type FlowmeterPositiveDisplacementProps =
  LabeledProps<Primitives.FlowmeterPositiveDisplacementProps>;
export const FlowmeterTurbine = createLabeled(Primitives.FlowmeterTurbine);
export type FlowmeterTurbineProps = LabeledProps<Primitives.FlowmeterTurbineProps>;
export const FlowmeterPulse = createLabeled(Primitives.FlowmeterPulse);
export type FlowmeterPulseProps = LabeledProps<Primitives.FlowmeterPulseProps>;
export const FlowmeterFloatSensor = createLabeled(Primitives.FlowmeterFloatSensor);
export type FlowmeterFloatSensorProps =
  LabeledProps<Primitives.FlowmeterFloatSensorProps>;
export const FlowmeterOrifice = createLabeled(Primitives.FlowmeterOrifice);
export type FlowmeterOrificeProps = LabeledProps<Primitives.FlowmeterOrificeProps>;
export const HeatExchangerGeneral = createLabeled(Primitives.HeatExchangerGeneral);
export type HeatExchangerGeneralProps =
  LabeledProps<Primitives.HeatExchangerGeneralProps>;
export const HeatExchangerM = createLabeled(Primitives.HeatExchangerM);
export type HeatExchangerMProps = LabeledProps<Primitives.HeatExchangerMProps>;
export const HeatExchangerStraightTube = createLabeled(
  Primitives.HeatExchangerStraightTube,
);
export type HeatExchangerStraightTubeProps =
  LabeledProps<Primitives.HeatExchangerStraightTubeProps>;
export const FlameArrestor = createLabeled(Primitives.FlameArrestor);
export type FlameArrestorProps = LabeledProps<Primitives.FlameArrestorProps>;
export const FlameArrestorExplosion = createLabeled(Primitives.FlameArrestorExplosion);
export type FlameArrestorExplosionProps =
  LabeledProps<Primitives.FlameArrestorExplosionProps>;
export const FlameArrestorDetonation = createLabeled(
  Primitives.FlameArrestorDetonation,
);
export type FlameArrestorDetonationProps =
  LabeledProps<Primitives.FlameArrestorDetonationProps>;
export const FlameArrestorFireRes = createLabeled(Primitives.FlameArrestorFireRes);
export type FlameArrestorFireResProps =
  LabeledProps<Primitives.FlameArrestorFireResProps>;
export const FlameArrestorFireResDetonation = createLabeled(
  Primitives.FlameArrestorFireResDetonation,
);
export type FlameArrestorFireResDetonationProps =
  LabeledProps<Primitives.FlameArrestorFireResDetonationProps>;
export const Thruster = createLabeled(Primitives.Thruster);
export type ThrusterProps = LabeledProps<Primitives.ThrusterProps>;
export const Nozzle = createLabeled(Primitives.Nozzle);
export type NozzleProps = LabeledProps<Primitives.NozzleProps>;
export const Strainer = createLabeled(Primitives.Strainer);
export type StrainerProps = LabeledProps<Primitives.StrainerProps>;
export const StrainerCone = createLabeled(Primitives.StrainerCone);
export type StrainerConeProps = LabeledProps<Primitives.StrainerConeProps>;

// ||||||||| TOGGLE DUMMY ||||||||
export const NeedleValve = createDummyToggle(Primitives.NeedleValve);
export type NeedleValveProps = DummyToggleProps<Primitives.NeedleValveProps>;
export const ReliefValve = createDummyToggle(Primitives.ReliefValve);
export type ReliefValveProps = DummyToggleProps<Primitives.ReliefValveProps>;
export const SpringLoadedReliefValve = createDummyToggle(
  Primitives.SpringLoadedReliefValve,
);
export type SpringLoadedReliefValveProps =
  DummyToggleProps<Primitives.SpringLoadedReliefValveProps>;
export const AngledSpringLoadedReliefValve = createDummyToggle(
  Primitives.AngledSpringLoadedReliefValve,
);
export type AngledSpringLoadedReliefValveProps =
  DummyToggleProps<Primitives.AngledSpringLoadedReliefValveProps>;
export const ManualValve = createDummyToggle(Primitives.ManualValve);
export type ManualValveProps = DummyToggleProps<Primitives.ManualValveProps>;
export const AngledReliefValve = createDummyToggle(Primitives.AngledReliefValve);
export type AngledReliefValveProps =
  DummyToggleProps<Primitives.AngledReliefValveProps>;
export const BreatherValve = createDummyToggle(Primitives.BreatherValve);
export type BreatherValveProps = DummyToggleProps<Primitives.BreatherValveProps>;

// ||||||||| CUSTOM ||||||||

export interface TankProps extends Omit<Primitives.TankProps, "boxBorderRadius"> {
  label?: LabelExtensionProps;
}

export const Tank = createLabeled(
  ({
    backgroundColor,
    onChange,
    orientation,
    color,
    dimensions,
    borderRadius,
  }: SymbolProps<TankProps>): ReactElement => (
    <Primitives.Tank
      onResize={(dims) => onChange({ dimensions: dims })}
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  ),
);

export const TankPreview = (props: TankProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 25, height: 50 }} />
);

export interface BoxProps extends Omit<TankProps, "borderRadius"> {
  borderRadius?: number;
  strokeWidth?: number;
}

export const Triangle = createLabeled(
  ({
    sideLength,
    rotation,
    color,
    backgroundColor,
    numSides,
    ...rest
  }: SymbolProps<Primitives.PolygonProps>) => (
    <Primitives.Polygon
      numSides={3}
      sideLength={sideLength}
      rotation={rotation}
      color={color}
      backgroundColor={backgroundColor}
      {...rest}
    />
  ),
);
export type TriangleProps = LabeledProps<Primitives.PolygonProps>;

export const PolygonSymbol = createLabeled(
  ({
    numSides,
    sideLength,
    cornerRounding,
    rotation,
    color,
    backgroundColor,
    strokeWidth,
    ...rest
  }: SymbolProps<Primitives.PolygonProps>) => (
    <Primitives.Polygon
      numSides={numSides}
      sideLength={sideLength}
      cornerRounding={cornerRounding}
      rotation={rotation}
      color={color}
      backgroundColor={backgroundColor}
      strokeWidth={strokeWidth}
      {...rest}
    />
  ),
);

export const Circle = createLabeled(
  ({
    radius,
    color,
    backgroundColor,
    strokeWidth,
    ...rest
  }: SymbolProps<Primitives.CircleShapeProps>) => (
    <Primitives.CircleShape
      radius={radius}
      color={color}
      backgroundColor={backgroundColor}
      strokeWidth={strokeWidth}
      {...rest}
    />
  ),
);

export const Box = createLabeled(
  ({
    backgroundColor,
    borderRadius,
    onChange,
    orientation,
    color,
    dimensions,
    strokeWidth,
  }: SymbolProps<BoxProps>): ReactElement => (
    <Primitives.Tank
      onResize={(dims) => onChange({ dimensions: dims })}
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      boxBorderRadius={borderRadius}
      backgroundColor={backgroundColor}
      strokeWidth={strokeWidth}
    />
  ),
);

export const BoxPreview = (props: BoxProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 25, height: 50 }} borderRadius={0} />
);

export interface SetpointProps
  extends Omit<Primitives.SetpointProps, "value" | "onChange">,
    Omit<CoreSetpoint.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Setpoint = ({
  label,
  symbolKey,
  orientation = "left",
  control,
  units,
  source,
  sink,
  color,
  onChange,
  selected,
  draggable,
  size,
  disabled,
}: SymbolProps<SetpointProps>): ReactElement => {
  const { value, set } = CoreSetpoint.use({ aetherKey: symbolKey, source, sink });
  const gridItems: GridItem[] = [];
  const controlItem = controlStateGridItem(control);
  if (controlItem != null) gridItems.push(controlItem);
  const labelItem = labelGridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);
  return (
    <Grid
      symbolKey={symbolKey}
      editable={selected && !draggable}
      onRotate={() =>
        onChange({
          orientation: location.rotate90(orientation),
        } as Partial<SetpointProps>)
      }
      items={gridItems}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } } as Partial<SetpointProps>);
      }}
    >
      <Primitives.Setpoint
        value={value}
        onChange={set}
        units={units}
        color={color}
        orientation={orientation}
        disabled={disabled}
        size={size}
      />
    </Grid>
  );
};

export const SetpointPreview = ({
  className,
  ...rest
}: SetpointProps): ReactElement => (
  <Primitives.Setpoint
    value={12}
    onChange={() => {}}
    units="mV"
    style={{ width: 120, transform: "scale(0.95)" }}
    className={CSS(CSS.BM("setpoint", "preview"), className)}
    disabled
    {...rest}
  >
    <Text.Text level="p">10.0</Text.Text>
  </Primitives.Setpoint>
);

export interface ValueProps
  extends Omit<CoreValue.UseProps, "box" | "aetherKey">,
    Primitives.ValueProps {
  position?: xy.XY;
  label?: LabelExtensionProps;
  color?: Color.Crude;
  textColor?: Color.Crude;
  tooltip?: string[];
}

export const Value = ({
  symbolKey,
  label,
  level = "p",
  position,
  textColor,
  color,
  telem,
  units,
  onChange,
  inlineSize = 70,
  selected,
  draggable,
  notation,
}: SymbolProps<ValueProps>): ReactElement => {
  const font = Theming.useTypography(level);
  const valueBoxHeight = (font.lineHeight + 0.5) * font.baseSize + 2;

  const { width: oWidth } = CoreValue.use({
    aetherKey: symbolKey,
    color: textColor,
    level,
    box: box.construct(xy.translateY({ ...position }, 1), {
      height: valueBoxHeight,
      width: inlineSize,
    }),
    telem,
    minWidth: inlineSize,
    notation,
  });

  const gridItems: GridItem[] = [];
  const labelItem = labelGridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);

  return (
    <Grid
      editable={selected && !draggable}
      symbolKey={symbolKey}
      items={gridItems}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({
          label: { ...label, orientation: loc },
        } as Partial<ValueProps>);
      }}
    >
      <Primitives.Value
        color={color}
        dimensions={{
          height: valueBoxHeight,
          width: oWidth,
        }}
        inlineSize={inlineSize}
        units={units}
        unitsLevel={Text.downLevel(level)}
      />
    </Grid>
  );
};

export const ValuePreview = ({ color }: ValueProps): ReactElement => (
  <Primitives.Value color={color} dimensions={{ width: 60, height: 25 }} units="psi">
    <Text.Text level="p">50.00</Text.Text>
  </Primitives.Value>
);

export interface ButtonProps
  extends Omit<Primitives.ButtonProps, "label" | "onClick">,
    Omit<CoreButton.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Button = ({
  symbolKey,
  label,
  orientation = "left",
  sink,
  control,
  selected,
  draggable,
  onChange,
  ...rest
}: SymbolProps<ButtonProps>) => {
  const { click } = CoreButton.use({ aetherKey: symbolKey, sink });
  const gridItems: GridItem[] = [];
  const controlItem = controlStateGridItem(control);
  if (controlItem != null) gridItems.push(controlItem);
  return (
    <Grid
      onRotate={() =>
        onChange({
          orientation: location.rotate90(orientation),
        } as Partial<ButtonProps>)
      }
      editable={selected && !draggable}
      symbolKey={symbolKey}
      items={gridItems}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } } as Partial<ButtonProps>);
      }}
    >
      <Primitives.Button
        {...label}
        onClick={click}
        orientation={orientation}
        {...rest}
      />
    </Grid>
  );
};

export const ButtonPreview = ({ label: _, ...rest }: ButtonProps): ReactElement => (
  <Primitives.Button label="Button" {...rest} />
);

export interface LightProps
  extends Primitives.LightProps,
    Omit<CoreLight.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const Light = ({
  symbolKey,
  label,
  source,
  onChange,
  selected,
  draggable,
  ...rest
}: SymbolProps<LightProps>): ReactElement => {
  const { enabled } = CoreLight.use({ aetherKey: symbolKey, source });
  const gridItems: GridItem[] = [];
  const labelItem = labelGridItem(label, onChange);
  if (labelItem != null) gridItems.push(labelItem);
  return (
    <Grid
      items={gridItems}
      editable={selected && !draggable}
      symbolKey={symbolKey}
      onLocationChange={(key, loc) => {
        if (key !== "label") return;
        onChange({ label: { ...label, orientation: loc } } as Partial<LightProps>);
      }}
    >
      <Primitives.Light enabled={enabled} {...rest} />
    </Grid>
  );
};

export const TextBoxPreview = (
  props: SymbolProps<Primitives.TextBoxProps>,
): ReactElement => <Primitives.TextBox {...props} autoFit text="Text Box" />;

export interface OffPageReferenceProps
  extends Omit<Primitives.OffPageReferenceProps, "label"> {
  label: LabelExtensionProps;
}

export const OffPageReference = ({
  label: { label, level },
  orientation,
  color,
  onChange,
}: SymbolProps<OffPageReferenceProps>): ReactElement => (
  <Primitives.OffPageReference
    onLabelChange={(label) => onChange({ label: { label, level } })}
    label={label}
    level={level}
    orientation={orientation}
    color={color}
  />
);

export const OffPageReferencePreview = ({
  label: _,
  ...rest
}: OffPageReferenceProps) => (
  <Primitives.OffPageReference label="Off Page" {...rest} orientation="right" />
);
export const Cylinder = createLabeled<
  SymbolProps<Omit<Primitives.CylinderProps, "onChange">>
>(
  ({
    backgroundColor,
    onChange,
    orientation,
    color,
    dimensions,
    borderRadius,
  }): ReactElement => (
    <Primitives.Cylinder
      onResize={(dimensions) => onChange({ dimensions })}
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  ),
);
export type CylinderProps = LabeledProps<Omit<Primitives.CylinderProps, "onChange">>;

export const CylinderPreview = (props: CylinderProps): ReactElement => (
  <Primitives.Cylinder {...props} dimensions={{ width: 25, height: 50 }} />
);
