import { ReactElement } from "react";

import { Handle, Position } from "reactflow";

import {
  PIDElementFormProps,
  StatefulPIDElementProps,
  PIDElementSpec,
} from "./PIDElement";

import { CSS, Input } from "@/core";
import { ValueLabeled, ValueLabeledProps } from "@/core/vis/Value/ValueLabeled";
import { Telem } from "@/telem";
import { RangeNumerictelemProps as RangeNumericTelemProps } from "@/telem/range/aether";
import { RangeNumericTelemForm } from "@/telem/range/forms";

import "@/vis/pid/ValuePIDElement.css";

export const ZERO_PROPS: ValuePIDElementProps = {
  label: "Value",
  telem: {
    channel: 0,
  },
  units: "psi",
  level: "p",
};

export interface ValuePIDElementProps extends Omit<ValueLabeledProps, "telem"> {
  telem: RangeNumericTelemProps;
}

const ValuePIDElement = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  className,
  ...props
}: StatefulPIDElementProps<ValuePIDElementProps>): ReactElement => {
  const telem = Telem.Range.useNumeric(pTelem);
  const onLabelChange = (label: string): void => {
    onChange({ ...props, label, telem: pTelem });
  };

  return (
    <>
      {editable && <Handle position={Position.Top} type="source" />}
      {editable && <Handle position={Position.Bottom} type="target" />}
      {editable && <Handle position={Position.Left} type="source" />}
      {editable && <Handle position={Position.Right} type="target" />}
      <ValueLabeled
        className={CSS(className, selected && CSS.BM("value-pid-element", "selected"))}
        {...props}
        telem={telem}
        onLabelChange={onLabelChange}
      />
    </>
  );
};

const ValuePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValuePIDElementProps>): ReactElement => {
  const handleTelemChange = (telem: RangeNumericTelemProps): void => {
    onChange({ ...value, telem });
  };

  const handleLabelChange = (label: string): void => {
    onChange({ ...value, label });
  };

  const handleUnitsChange = (units: string): void => {
    onChange({ ...value, units });
  };

  return (
    <>
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />
      <Input.Item<string>
        label="Units"
        value={value.units}
        onChange={handleUnitsChange}
      />
      <RangeNumericTelemForm value={value.telem} onChange={handleTelemChange} />;
    </>
  );
};

const ValuePIDElementPreview = (): ReactElement => {
  const telem = Telem.Static.useNumeric(500);
  return <ValueLabeled label="Value" units="psi" telem={telem} level="p" />;
};

export const ValuePIDElementSpec: PIDElementSpec<ValuePIDElementProps> = {
  type: "value",
  title: "Value",
  initialProps: ZERO_PROPS,
  Element: ValuePIDElement,
  Form: ValuePIDElementForm,
  Preview: ValuePIDElementPreview,
};
