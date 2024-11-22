// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/Symbols.css";

import {
  box,
  dimensions,
  direction,
  location,
  type UnknownRecord,
  xy,
} from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import { width } from "node_modules/@synnaxlabs/x/dist/src/spatial/box/box";
import {
  Fragment,
  PropsWithChildren,
  type ReactElement,
  ReactNode,
  useCallback,
  useState,
} from "react";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { useResize, useSyncedRef } from "@/hooks";
import { Control } from "@/telem/control";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Button as CoreButton } from "@/vis/button";
import { Light as CoreLight } from "@/vis/light";
import { Labeled, type LabelExtensionProps } from "@/vis/schematic/Labeled";
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

const swapXLocation = (l: location.Outer): location.Outer =>
  direction.construct(l) === "x" ? (location.swap(l) as location.Outer) : l;
const swapYLocation = (l: location.Outer): location.Outer =>
  direction.construct(l) === "y" ? (location.swap(l) as location.Outer) : l;

interface GridItem {
  key: string;
  element: ReactNode;
  location: location.Outer;
}

interface GridProps extends PropsWithChildren<{}> {
  items: GridItem[];
}

interface GridElProps {
  items: GridItem[];
  loc: location.Outer;
}

const GridEl = ({ items: fItems, loc }: GridElProps): ReactElement | null => {
  const items = fItems.filter((i) => i.location === loc);
  if (items.length === 0) return null;
  return (
    <Align.Space
      direction={location.direction(loc)}
      className={CSS(CSS.BE("grid", "item"), CSS.loc(loc))}
    >
      {items.map(({ element, key }) => (
        <Fragment key={key}>{element}</Fragment>
      ))}
    </Align.Space>
  );
};

const Grid = ({ children, items }: GridProps) => {
  return (
    <div className={CSS(CSS.B("symbol-grid"))}>
      <GridEl items={items} loc="top" />
      <GridEl items={items} loc="left" />
      <GridEl items={items} loc="right" />
      <GridEl items={items} loc="bottom" />
      {children}
    </div>
  );
};

const ControlState = ({
  showChip = true,
  showIndicator = true,
  indicator,
  orientation = "left",
  chip,
  children,
  show = true,
  ...props
}: ControlStateProps): ReactElement => (
  <Align.Space
    // direction={location.rotate90(orientation)}
    align="center"
    justify="center"
    empty
    {...props}
  >
    {children}
  </Align.Space>
);

export type SymbolProps<P extends object = UnknownRecord> = P & {
  symbolKey: string;
  position: xy.XY;
  aetherKey: string;
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
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
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

const labelGridItem = (props?: LabelExtensionProps): GridItem | null => {
  if (props == null) return null;
  const { label, level = "p", orientation = "top", direction } = props;
  if (label == null || label.length === 0) return null;
  return {
    key: "label",
    element: (
      <Text.Editable
        className={CSS(CSS.BE("symbol", "label"), CSS.dir(direction))}
        level={level}
        value={label}
      />
    ),
    location: orientation,
  };
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

export const SolenoidValve = Aether.wrap<SymbolProps<SolenoidValveProps>>(
  "SolenoidValve",
  ({
    aetherKey,
    label,
    onChange,
    position: _,
    orientation = "left",
    normallyOpen,
    source,
    sink,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    const gridItems: GridItem[] = [];
    const labelItem = labelGridItem(label);
    if (labelItem != null) gridItems.push(labelItem);
    const controlItem = controlStateGridItem(control);
    if (controlItem != null) gridItems.push(controlItem);
    console.log(gridItems);
    return (
      <Grid items={gridItems}>
        <Primitives.SolenoidValve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          orientation={orientation}
          normallyOpen={normallyOpen}
          {...rest}
        />
      </Grid>
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

export const Tank = ({
  backgroundColor,
  label,
  onChange,
  orientation,
  color,
  dimensions,
  borderRadius,
}: SymbolProps<TankProps>): ReactElement => (
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
);

export const TankPreview = (props: TankProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 25, height: 50 }} />
);

export interface BoxProps extends Omit<TankProps, "borderRadius"> {
  borderRadius?: number;
}

export const Box = ({
  backgroundColor,
  borderRadius,
  label,
  onChange,
  orientation,
  color,
  dimensions,
}: SymbolProps<BoxProps>): ReactElement => (
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
  aetherKey: _,
  ...rest
}: SymbolProps<ReliefValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ReliefValve {...rest} />
  </Labeled>
);

export const ReliefValvePreview = (props: ReliefValveProps): ReactElement => (
  <Primitives.ReliefValve {...props} />
);

export interface SpringLoadedReliefValveProps
  extends Primitives.SpringLoadedReliefValveProps {
  label?: LabelExtensionProps;
}

export const SpringLoadedReliefValve = ({
  label,
  onChange,
  aetherKey: _,
  ...rest
}: SymbolProps<SpringLoadedReliefValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.SpringLoadedReliefValve {...rest} />
  </Labeled>
);

export const SpringLoadedReliefValvePreview = (
  props: SpringLoadedReliefValveProps,
): ReactElement => <Primitives.SpringLoadedReliefValve {...props} />;

export interface AngledSpringLoadedReliefValveProps
  extends Primitives.AngledSpringLoadedReliefValveProps {
  label?: LabelExtensionProps;
}

export const AngledSpringLoadedReliefValve = ({
  label,
  onChange,
  aetherKey: _,
  ...rest
}: SymbolProps<AngledSpringLoadedReliefValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.AngledSpringLoadedReliefValve {...rest} />
  </Labeled>
);

export const AngledSpringLoadedReliefValvePreview = (
  props: AngledSpringLoadedReliefValveProps,
): ReactElement => <Primitives.AngledSpringLoadedReliefValve {...props} />;

export interface RegulatorProps extends Primitives.RegulatorProps {
  label?: LabelExtensionProps;
}

export const Regulator = ({
  label,
  onChange,
  aetherKey: _,
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
  aetherKey: _,
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
  aetherKey: _,
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
  aetherKey,
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

export interface ISOCapProps extends Primitives.ISOCapProps {
  label?: LabelExtensionProps;
}

export const ISOCap = ({
  label,
  aetherKey,
  onChange,
  ...rest
}: SymbolProps<ISOCapProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ISOCap {...rest} />
  </Labeled>
);

export const ISOCapPreview = (props: ISOCapProps): ReactElement => (
  <Primitives.ISOCap {...props} />
);

export interface ManualValveProps extends Primitives.ManualValveProps {
  label?: LabelExtensionProps;
}

export const ManualValve = ({
  label,
  aetherKey,
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
    color,
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
            color={color}
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
  aetherKey,
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
  aetherKey: _,
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
  aetherKey: _,
  ...rest
}: SymbolProps<CheckValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.CheckValve {...rest} />
  </Labeled>
);

export const CheckValvePreview = (props: CheckValveProps): ReactElement => (
  <Primitives.CheckValve {...props} />
);

export interface ISOCheckValveProps extends Primitives.ISOCheckValveProps {
  label?: LabelExtensionProps;
}

export const ISOCheckValve = ({
  label,
  onChange,
  aetherKey: _,
  ...rest
}: SymbolProps<ISOCheckValveProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ISOCheckValve {...rest} />
  </Labeled>
);

export const ISOCheckValvePreview = (props: ISOCheckValveProps): ReactElement => (
  <Primitives.ISOCheckValve {...props} />
);

export interface OrificeProps extends Primitives.OrificeProps {
  label?: LabelExtensionProps;
}

export const Orifice = ({
  label,
  onChange,
  aetherKey: _,
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
  aetherKey: _,
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
    inlineSize,
    notation,
  }): ReactElement => {
    const font = Theming.useTypography(level);
    const valueBoxHeight = (font.lineHeight + 0.5) * font.baseSize + 2;

    const { width: oWidth } = CoreValue.use({
      aetherKey,
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
          onChange={onChange}
          {...label}
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
        </Labeled>
      </Tooltip.Dialog>
    );
  },
);

export const ValuePreview = ({ color }: ValueProps): ReactElement => (
  <Primitives.Value color={color} dimensions={{ width: 60, height: 25 }} units={"psi"}>
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

export const StaticMixer = ({
  label,
  onChange,
  aetherKey,
  ...rest
}: SymbolProps<StaticMixerProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.StaticMixer {...rest} />
  </Labeled>
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

export interface AgitatorProps
  extends Primitives.AgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Agitator = Aether.wrap<SymbolProps<AgitatorProps>>(
  "agitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.Agitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const AgitatorPreview = (props: AgitatorProps): ReactElement => (
  <Primitives.Agitator {...props} />
);

export interface PropellerAgitatorProps
  extends Primitives.PropellerAgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const PropellerAgitator = Aether.wrap<SymbolProps<PropellerAgitatorProps>>(
  "propellerAgitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.PropellerAgitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const PropellerAgitatorPreview = (
  props: PropellerAgitatorProps,
): ReactElement => <Primitives.PropellerAgitator {...props} />;

export interface FlatBladeAgitatorProps
  extends Primitives.FlatBladeAgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const FlatBladeAgitator = Aether.wrap<SymbolProps<FlatBladeAgitatorProps>>(
  "flatBladeAgitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.FlatBladeAgitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const FlatBladeAgitatorPreview = (
  props: FlatBladeAgitatorProps,
): ReactElement => <Primitives.FlatBladeAgitator {...props} />;

export interface PaddleAgitatorProps
  extends Primitives.PaddleAgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const PaddleAgitator = Aether.wrap<SymbolProps<PaddleAgitatorProps>>(
  "paddleAgitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.PaddleAgitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const PaddleAgitatorPreview = (props: PaddleAgitatorProps): ReactElement => (
  <Primitives.PaddleAgitator {...props} />
);

export interface CrossBeamAgitatorProps
  extends Primitives.CrossBeamAgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const CrossBeamAgitator = Aether.wrap<SymbolProps<CrossBeamAgitatorProps>>(
  "crossBeamAgitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.CrossBeamAgitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const CrossBeamAgitatorPreview = (
  props: CrossBeamAgitatorProps,
): ReactElement => <Primitives.CrossBeamAgitator {...props} />;

export interface HelicalAgitatorProps
  extends Primitives.HelicalAgitatorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const HelicalAgitator = Aether.wrap<SymbolProps<HelicalAgitatorProps>>(
  "helicalAgitator",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.HelicalAgitator
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const HelicalAgitatorPreview = (props: HelicalAgitatorProps): ReactElement => (
  <Primitives.HelicalAgitator {...props} />
);

export interface CompressorProps
  extends Primitives.CompressorProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Compressor = Aether.wrap<SymbolProps<CompressorProps>>(
  "compressor",
  ({
    aetherKey,
    label,
    orientation = "left",
    source,
    sink,
    onChange,
    control,
    ...rest
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <ControlState orientation={orientation} {...control}>
          <Primitives.Compressor
            enabled={enabled}
            orientation={orientation}
            triggered={triggered}
            onClick={toggle}
            {...rest}
          />
        </ControlState>
      </Labeled>
    );
  },
);

export const CompressorPreview = (props: CompressorProps): ReactElement => (
  <Primitives.Compressor {...props} />
);

export interface TextBoxProps extends Primitives.TextBoxProps {}

export const TextBox = (props: SymbolProps<TextBoxProps>): ReactElement => (
  <Primitives.TextBox {...props} />
);

export const TextBoxPreview = ({
  level = "p",
  width = 100,
  ...rest
}: SymbolProps<TextBoxProps>): ReactElement => (
  <Primitives.TextBox
    className={CSS.B("symbol")}
    level={level}
    width={width}
    {...rest}
  />
);

export interface OffPageReferenceProps
  extends Omit<Primitives.OffPageReferenceProps, "label"> {
  label: LabelExtensionProps;
}

export interface VentProps extends Primitives.VentProps {
  label?: LabelExtensionProps;
}

export const Vent = ({
  label,
  aetherKey,
  onChange,
  ...rest
}: SymbolProps<VentProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Vent {...rest} />
  </Labeled>
);

export const VentPreview = (props: VentProps): ReactElement => (
  <Primitives.Vent {...props} />
);

export const OffPageReference = ({
  label: { label, level },
  aetherKey,
  position: _,
  ...props
}: SymbolProps<OffPageReferenceProps>): ReactElement => (
  <Primitives.OffPageReference
    label={label}
    level={level}
    {...props}
    className={CSS.B("symbol")}
  />
);

export const OffPageReferencePreview = ({
  label: _,
  ...props
}: OffPageReferenceProps) => (
  <Primitives.OffPageReference label="Off Page" {...props} orientation="right" />
);

export interface OrificePlateProps extends Primitives.OrificePlateProps {
  label?: LabelExtensionProps;
}

export const OrificePlate = ({
  label,
  aetherKey,
  onChange,
  ...rest
}: SymbolProps<OrificePlateProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.OrificePlate {...rest} />
  </Labeled>
);

export const OrificePlatePreview = (props: OrificePlateProps): ReactElement => (
  <Primitives.OrificePlate {...props} />
);

export interface ISOFilterProps extends Primitives.ISOFilterProps {
  label?: LabelExtensionProps;
}

export const ISOFilter = ({
  label,
  aetherKey,
  onChange,
  ...rest
}: SymbolProps<ISOFilterProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ISOFilter {...rest} />
  </Labeled>
);

export const ISOFilterPreview = (props: ISOFilterProps): ReactElement => (
  <Primitives.ISOFilter {...props} />
);

export interface CylinderProps
  extends Omit<Primitives.CylinderProps, "boxBorderRadius"> {
  label?: LabelExtensionProps;
}

export const Cylinder = ({
  backgroundColor,
  label,
  onChange,
  orientation,
  color,
  dimensions,
  borderRadius,
}: SymbolProps<CylinderProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.Cylinder
      onResize={(dims) => onChange({ dimensions: dims })}
      orientation={orientation}
      color={color}
      dimensions={dimensions}
      borderRadius={borderRadius}
      backgroundColor={backgroundColor}
    />
  </Labeled>
);

export const CylinderPreview = (props: CylinderProps): ReactElement => (
  <Primitives.Cylinder {...props} dimensions={{ width: 25, height: 50 }} />
);

export interface ISOBurstDiscProps extends Primitives.ISOBurstDiscProps {
  label?: LabelExtensionProps;
}

export const ISOBurstDisc = ({
  label,
  onChange,
  aetherKey: _,
  ...rest
}: SymbolProps<ISOBurstDiscProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.ISOBurstDisc {...rest} />
  </Labeled>
);

export const ISOBurstDiscPreview = (props: ISOBurstDiscProps): ReactElement => (
  <Primitives.ISOBurstDisc {...props} />
);

export interface TJunctionProps extends Primitives.TJunctionProps {
  label?: LabelExtensionProps;
}

export const TJunction = ({
  label,
  aetherKey,
  onChange,
  ...rest
}: SymbolProps<TJunctionProps>): ReactElement => (
  <Labeled {...label} onChange={onChange}>
    <Primitives.TJunction {...rest} />
  </Labeled>
);

export const TJunctionPreview = (props: TJunctionProps): ReactElement => (
  <Primitives.TJunction {...props} />
);
