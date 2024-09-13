// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/Symbols.css";

import { box, direction, location, type UnknownRecord, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { useResize } from "@/hooks";
import { Control } from "@/telem/control";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Button as CoreButton } from "@/vis/button";
import { useInitialViewport } from "@/vis/diagram/aether/Diagram";
import { Light as CoreLight } from "@/vis/light";
import { Labeled, type LabelExtensionProps } from "@/vis/schematic/Labeled";
import { Primitives } from "@/vis/schematic/primitives";
import { Setpoint as CoreSetpoint } from "@/vis/setpoint";
import { Toggle } from "@/vis/toggle";
import { Value as CoreValue } from "@/vis/value";

export interface ControlStateProps extends Omit<Align.SpaceProps, "direction"> {
  showChip?: boolean;
  showIndicator?: boolean;
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
  orientation?: location.Outer;
}

const swapXLocation = (l: location.Outer): location.Outer =>
  direction.construct(l) === "x" ? (location.swap(l) as location.Outer) : l;
const swapYLocation = (l: location.Outer): location.Outer =>
  direction.construct(l) === "y" ? (location.swap(l) as location.Outer) : l;

const ControlState = ({
  showChip = true,
  showIndicator = true,
  indicator,
  orientation = "left",
  chip,
  children,
  ...props
}: ControlStateProps): ReactElement => (
  <Align.Space
    direction={location.rotate90(orientation)}
    align="center"
    justify="center"
    empty
    {...props}
  >
    <Align.Space
      direction={direction.construct(orientation)}
      align="center"
      className={CSS(CSS.B("control-state"))}
      size="small"
    >
      {showChip && <Control.Chip size="small" {...chip} />}
      {showIndicator && <Control.Indicator {...indicator} />}
    </Align.Space>
    {children}
  </Align.Space>
);

export type SymbolProps<P extends object = UnknownRecord> = P & {
  symbolKey: string;
  position: xy.XY;
  selected: boolean;
  onChange: (value: P) => void;
};

export interface ThreeWayValveProps
  extends Primitives.ThreeWayValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const ThreeWayValve = Aether.wrap<SymbolProps<ThreeWayValveProps>>(
  "ThreeWayValve",
  ({
    aetherKey,
    label,
    onChange,
    control,
    source,
    sink,
    orientation = "left",
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({
      aetherKey,
      source,
      sink,
    });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={swapXLocation(orientation)}>
          <Primitives.ThreeWayValve
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const ThreeWayValvePreview = (props: ThreeWayValveProps): ReactElement => (
  <Primitives.ThreeWayValve {...props} />
);

export interface ValveProps
  extends Primitives.ValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Valve = Aether.wrap<SymbolProps<ValveProps>>(
  "Valve",
  ({
    control,
    aetherKey,
    label,
    onChange,
    source,
    sink,
    orientation,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.Valve
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const ValvePreview = (props: ValveProps): ReactElement => (
  <Primitives.Valve {...props} />
);

export interface SolenoidValveProps
  extends Primitives.SolenoidValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const SolenoidValve = Aether.wrap<SymbolProps<SolenoidValveProps>>(
  "SolenoidValve",
  ({
    aetherKey,
    label,
    onChange,
    orientation = "left",
    normallyOpen,
    source,
    sink,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={swapYLocation(orientation)}>
          <Primitives.SolenoidValve
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            normallyOpen={normallyOpen}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const SolenoidValvePreview = (props: SolenoidValveProps): ReactElement => (
  <Primitives.SolenoidValve {...props} />
);

export interface FourWayValveProps
  extends Primitives.FourWayValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const FourWayValve = Aether.wrap<SymbolProps<FourWayValveProps>>(
  "FourWayValve",
  ({
    aetherKey,
    control,
    label,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.FourWayValve
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const FourWayValvePreview = (props: FourWayValveProps): ReactElement => (
  <Primitives.FourWayValve {...props} />
);

export interface AngledValveProps
  extends Primitives.AngledValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const AngledValve = Aether.wrap<SymbolProps<AngledValveProps>>(
  "AngleValve",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation = "left",
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={swapXLocation(orientation)}>
          <Primitives.AngledValve
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const AngledValvePreview = (props: AngledValveProps): ReactElement => (
  <Primitives.AngledValve {...props} />
);

export interface PumpProps
  extends Primitives.PumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Pump = Aether.wrap<SymbolProps<PumpProps>>(
  "Pump",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.Pump
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const PumpPreview = (props: PumpProps): ReactElement => (
  <Primitives.Pump {...props} />
);

export interface TankProps extends Omit<Primitives.TankProps, "boxBorderRadius"> {
  label?: LabelExtensionProps;
}

export const Tank = Aether.wrap<SymbolProps<TankProps>>(
  "Tank",
  ({
    backgroundColor,
    label,
    onChange,
    orientation,
    color,
    dimensions,
    borderRadius,
  }): ReactElement => (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Tank
        onResize={(dims) => onChange({ dimensions: dims })}
        orientation={orientation}
        color={color}
        dimensions={dimensions}
        borderRadius={borderRadius}
        backgroundColor={backgroundColor}
      />
    </Labeled>
  ),
);

export const TankPreview = (props: TankProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 25, height: 50 }} />
);

export interface BoxProps extends Omit<TankProps, "borderRadius"> {
  borderRadius?: number;
}

export const Box = Aether.wrap<SymbolProps<BoxProps>>(
  "Box",
  ({
    backgroundColor,
    borderRadius,
    label,
    onChange,
    orientation,
    color,
    dimensions,
  }): ReactElement => (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Tank
        onResize={(dims) => onChange({ dimensions: dims })}
        orientation={orientation}
        color={color}
        dimensions={dimensions}
        boxBorderRadius={borderRadius}
        backgroundColor={backgroundColor}
      />
    </Labeled>
  ),
);

export const BoxPreview = (props: BoxProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 25, height: 50 }} borderRadius={0} />
);

export interface ReliefValveProps extends Primitives.ReliefValveProps {
  label?: LabelExtensionProps;
}

export const ReliefValve = ({
  label,
  onChange,
  ...rest
}: SymbolProps<ReliefValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ReliefValve {...rest} />
  </Labeled>
);

export const ReliefValvePreview = (props: ReliefValveProps): ReactElement => (
  <Primitives.ReliefValve {...props} />
);

export interface RegulatorProps extends Primitives.RegulatorProps {
  label?: LabelExtensionProps;
}

export const Regulator = ({
  label,
  onChange,
  ...rest
}: SymbolProps<RegulatorProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Regulator {...rest} />
  </Labeled>
);

export const RegulatorPreview = (props: RegulatorProps): ReactElement => (
  <Primitives.Regulator {...props} />
);

export interface ElectricRegulatorProps extends Primitives.ElectricRegulatorProps {
  label?: LabelExtensionProps;
}

export const ElectricRegulator = ({
  label,
  onChange,
  ...rest
}: SymbolProps<ElectricRegulatorProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ElectricRegulator {...rest} />
  </Labeled>
);

export const ElectricRegulatorPreview = (
  props: ElectricRegulatorProps,
): ReactElement => <Primitives.ElectricRegulator {...props} />;

export interface BurstDiscProps extends Primitives.BurstDiscProps {
  label?: LabelExtensionProps;
}

export const BurstDisc = ({
  label,
  onChange,
  ...rest
}: SymbolProps<BurstDiscProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.BurstDisc {...rest} />
  </Labeled>
);

export const BurstDiscPreview = (props: BurstDiscProps): ReactElement => (
  <Primitives.BurstDisc {...props} />
);

export interface CapProps extends Primitives.CapProps {
  label?: LabelExtensionProps;
}

export const Cap = ({
  label,
  onChange,
  ...rest
}: SymbolProps<CapProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Cap {...rest} />
  </Labeled>
);

export const CapPreview = (props: CapProps): ReactElement => (
  <Primitives.Cap {...props} />
);

export interface ManualValveProps extends Primitives.ManualValveProps {
  label?: LabelExtensionProps;
}

export const ManualValve = ({
  label,
  onChange,
  ...rest
}: SymbolProps<ManualValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ManualValve {...rest} />
  </Labeled>
);

export const ManualValvePreview = (props: ManualValveProps): ReactElement => (
  <Primitives.ManualValve {...props} />
);

export interface SetpointProps
  extends Omit<Primitives.SetpointProps, "value" | "onChange">,
    Omit<CoreSetpoint.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Setpoint = Aether.wrap<SymbolProps<SetpointProps>>(
  "Setpoint",
  ({
    label,
    aetherKey,
    orientation,
    control,
    units,
    source,
    sink,
    onChange,
  }): ReactElement => {
    const { value, set } = CoreSetpoint.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState
          {...control}
          className={CSS.B("symbol")}
          orientation={orientation}
        >
          <Primitives.Setpoint
            value={value}
            onChange={set}
            units={units}
            orientation={orientation}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const SetpointPreview = (props: SetpointProps): ReactElement => (
  <Primitives.Setpoint value={12} onChange={() => {}} units={"mV"} {...props}>
    <Text.Text level="p">10.0</Text.Text>
  </Primitives.Setpoint>
);

export interface FilterProps extends Primitives.FilterProps {
  label?: LabelExtensionProps;
}

export const Filter = ({
  label,
  onChange,
  ...rest
}: SymbolProps<FilterProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Filter {...rest} />
  </Labeled>
);

export const FilterPreview = (props: FilterProps): ReactElement => (
  <Primitives.Filter {...props} />
);

export interface NeedleValveProps extends Primitives.NeedleValveProps {
  label?: LabelExtensionProps;
}

export const NeedleValve = ({
  label,
  onChange,
  ...rest
}: SymbolProps<NeedleValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.NeedleValve {...rest} />
  </Labeled>
);

export const NeedleValvePreview = (props: NeedleValveProps): ReactElement => (
  <Primitives.NeedleValve {...props} />
);

export interface CheckValveProps extends Primitives.CheckValveProps {
  label?: LabelExtensionProps;
}

export const CheckValve = ({
  label,
  onChange,
  ...rest
}: SymbolProps<CheckValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.CheckValve {...rest} />
  </Labeled>
);

export const CheckValvePreview = (props: CheckValveProps): ReactElement => (
  <Primitives.CheckValve {...props} />
);

export interface OrificeProps extends Primitives.OrificeProps {
  label?: LabelExtensionProps;
}

export const Orifice = ({
  label,
  onChange,
  ...rest
}: SymbolProps<OrificeProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Orifice {...rest} />
  </Labeled>
);

export const OrificePreview = (props: OrificeProps): ReactElement => (
  <Primitives.Orifice {...props} />
);

export interface AngledReliefValveProps extends Primitives.AngledReliefValveProps {
  label?: LabelExtensionProps;
}

export const AngledReliefValve = ({
  label,
  onChange,
  ...rest
}: SymbolProps<AngledReliefValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.AngledReliefValve {...rest} />
  </Labeled>
);

export const AngledReliefValvePreview = (
  props: Primitives.AngledReliefValveProps,
): ReactElement => <Primitives.AngledReliefValve {...props} />;

export interface ValueProps
  extends Omit<CoreValue.UseProps, "box" | "aetherKey">,
    Primitives.ValueProps {
  position?: xy.XY;
  label?: LabelExtensionProps;
  color?: Color.Crude;
  textColor?: Color.Crude;
  tooltip?: string[];
}

interface ValueDimensionsState {
  outerBox: box.Box;
  labelBox: box.Box;
}

export const Value = Aether.wrap<SymbolProps<ValueProps>>(
  "Value",
  ({
    aetherKey,
    label,
    level = "p",
    position,
    className,
    textColor,
    color,
    telem,
    units,
    onChange,
    tooltip,
  }): ReactElement => {
    const font = Theming.useTypography(level);
    const [dimensions, setDimensions] = useState<ValueDimensionsState>({
      outerBox: box.ZERO,
      labelBox: box.ZERO,
    });

    const valueBoxHeight = (font.lineHeight + 0.5) * font.baseSize + 2;
    const resizeRef = useResize(
      useCallback((b) => {
        // Find the element with the class pluto-symbol__label that is underneath
        // the 'react-flow__node' with the data-id of aetherKey
        const label = document.querySelector(
          `.react-flow__node[data-id="${aetherKey}"] .pluto-symbol__label`,
        );
        let labelBox = { ...box.ZERO };
        if (label != null) {
          labelBox = box.construct(label);
          labelBox = box.resize(labelBox, {
            width: box.width(labelBox),
            height: box.height(labelBox),
          });
        }
        setDimensions({ outerBox: b, labelBox });
      }, []),
      {},
    );

    const { zoom } = useInitialViewport();

    const adjustedBox = adjustBox({
      labelOrientation: label?.orientation ?? "top",
      hasLabel: label?.label != null && label?.label.length > 0,
      valueBoxHeight,
      position,
      zoom,
      ...dimensions,
    });

    const { width: oWidth } = CoreValue.use({
      aetherKey,
      color: textColor,
      level,
      box: adjustedBox,
      telem,
      minWidth: 60,
    });

    return (
      <Tooltip.Dialog
        location={{ y: "top" }}
        hide={tooltip == null || tooltip.length === 0}
      >
        <Align.Space direction="y">
          {tooltip?.map((t, i) => (
            <Text.Text key={i} level="small">
              {t}
            </Text.Text>
          ))}
        </Align.Space>
        <Labeled
          className={CSS(className, CSS.B("value-labeled"))}
          ref={resizeRef}
          onChange={onChange}
          {...label}
        >
          <Primitives.Value
            color={color}
            dimensions={{
              height: valueBoxHeight,
              width: oWidth,
            }}
            units={units}
          />
        </Labeled>
      </Tooltip.Dialog>
    );
  },
);

interface AdjustBoxProps {
  labelOrientation: location.Outer;
  zoom: number;
  outerBox: box.Box;
  labelBox: box.Box;
  valueBoxHeight: number;
  position: xy.XY;
  hasLabel: boolean;
}

const LABEL_SCALE = 0.9;

const adjustBox = ({
  labelOrientation,
  outerBox,
  labelBox,
  valueBoxHeight,
  position,
  hasLabel,
  zoom,
}: AdjustBoxProps): box.Box => {
  const labelDims = xy.scale(box.dims(labelBox), 1 / (LABEL_SCALE * zoom));
  const dir = direction.construct(labelOrientation);
  if (dir === "x")
    position = xy.translate(
      position,
      "y",
      Math.max((labelDims.y - valueBoxHeight) / 2 - 1, 0),
    );
  if (hasLabel && labelOrientation === "left")
    position = xy.translate(position, "x", labelDims.x + 4);
  else if (hasLabel && labelOrientation === "top")
    position = xy.translate(position, "y", labelDims.y + 4);
  return box.construct(position.x, position.y, box.width(outerBox), valueBoxHeight);
};

export const ValuePreview = ({ color }: ValueProps): ReactElement => (
  <Primitives.Value
    color={color}
    dimensions={{
      width: 60,
      height: 25,
    }}
    units={"psi"}
  >
    <Text.Text level="p">50.00</Text.Text>
  </Primitives.Value>
);

export interface SwitchProps
  extends Primitives.SwitchProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Switch = Aether.wrap<SymbolProps<SwitchProps>>(
  "Switch",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.Switch
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const SwitchPreview = (props: SwitchProps): ReactElement => (
  <Primitives.Switch {...props} />
);

export interface ButtonProps
  extends Omit<Primitives.ButtonProps, "label" | "onClick">,
    Omit<CoreButton.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Button = Aether.wrap<SymbolProps<ButtonProps>>(
  "Button",
  ({ aetherKey, label, orientation, sink, control, ...rest }) => {
    const { click } = CoreButton.use({ aetherKey, sink });
    return (
      <ControlState {...control} className={CSS.B("symbol")} orientation={orientation}>
        <Primitives.Button {...label} onClick={click} {...rest} />
      </ControlState>
    );
  },
);

export const ButtonPreview = ({ label: _, ...props }: ButtonProps): ReactElement => (
  <Primitives.Button label="Button" {...props} />
);

export interface ScrewPumpProps
  extends Primitives.ScrewPumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const ScrewPump = Aether.wrap<SymbolProps<ScrewPumpProps>>(
  "screwPump",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.ScrewPump
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const ScrewPumpPreview = (props: ScrewPumpProps): ReactElement => (
  <Primitives.ScrewPump {...props} />
);

export interface VacuumPumpProps
  extends Primitives.VacuumPumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const VacuumPump = Aether.wrap<SymbolProps<VacuumPumpProps>>(
  "vacuumPump",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.VacuumPump
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const VacuumPumpPreview = (props: VacuumPumpProps): ReactElement => (
  <Primitives.VacuumPump {...props} />
);

export interface CavityPumpProps
  extends Primitives.CavityPumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const CavityPump = Aether.wrap<SymbolProps<CavityPumpProps>>(
  "progressiveCavityPump",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.CavityPump
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const CavityPumpPreview = (props: CavityPumpProps): ReactElement => (
  <Primitives.CavityPump {...props} />
);

export interface PistonPumpProps
  extends Primitives.PistonPumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const PistonPump = Aether.wrap<SymbolProps<PistonPumpProps>>(
  "pistonPump",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.PistonPump
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const PistonPumpPreview = (props: PistonPumpProps): ReactElement => (
  <Primitives.PistonPump {...props} />
);

export interface StaticMixerProps extends Primitives.StaticMixerProps {
  label?: LabelExtensionProps;
}

export const StaticMixer = Aether.wrap<SymbolProps<StaticMixerProps>>(
  "statixMixer",
  ({ label, onChange, ...rest }): ReactElement => (
    <Labeled {...label} onChange={onChange}>
      <Primitives.StaticMixer {...rest} />
    </Labeled>
  ),
);

export const StaticMixerPreview = (props: StaticMixerProps): ReactElement => (
  <Primitives.StaticMixer {...props} />
);

export interface RotaryMixerProps
  extends Primitives.RotaryMixerProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const RotaryMixer = Aether.wrap<SymbolProps<RotaryMixerProps>>(
  "rotaryMixer",
  ({
    aetherKey,
    label,
    control,
    onChange,
    orientation,
    source,
    sink,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState {...control} orientation={orientation}>
          <Primitives.RotaryMixer
            enabled={enabled}
            triggered={triggered}
            onClick={toggle}
            orientation={orientation}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const RotaryMixerPreview = (props: RotaryMixerProps): ReactElement => (
  <Primitives.RotaryMixer {...props} />
);

export interface LightProps
  extends Primitives.LightProps,
    Omit<CoreLight.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const Light = Aether.wrap<SymbolProps<LightProps>>(
  "light",
  ({ aetherKey, label, source, onChange, ...rest }): ReactElement => {
    const { enabled } = CoreLight.use({ aetherKey, source });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.Light enabled={enabled} {...rest} />
      </Labeled>
    );
  },
);

export const LightPreview = (props: LightProps): ReactElement => (
  <Primitives.Light {...props} />
);
