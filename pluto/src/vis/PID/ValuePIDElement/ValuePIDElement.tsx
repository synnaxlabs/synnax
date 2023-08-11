// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Handle, Position } from "reactflow";

import { CSS, Input } from "@/core";
import { ValueLabeled, ValueLabeledProps } from "@/core/vis/Value/ValueLabeled";
import { Telem } from "@/telem";
import { RemoteTelemNumericProps, RemoteTelem } from "@/telem/remote/main";
import {
  PIDElementFormProps,
  StatefulPIDElementProps,
  PIDElementSpec,
} from "@/vis/PID/PIDElement";

import "@/vis/PID/ValuePIDElement/ValuePIDElement.css";

export const ZERO_PROPS: ValuePIDElementProps = {
  label: "Value",
  telem: {
    channel: 0,
  },
  units: "psi",
  level: "p",
};

export interface ValuePIDElementProps extends Omit<ValueLabeledProps, "telem"> {
  telem: RemoteTelemNumericProps;
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
      {editable && (
        <>
          <Handle position={Position.Top} type="source" id="top" />
          <Handle position={Position.Left} type="target" id="left" />
          <Handle position={Position.Right} type="source" id="right" />
          <Handle position={Position.Bottom} type="target" id="bottom" />
        </>
      )}
      <ValueLabeled
        className={CSS(className, CSS.B("value-pid-element"), CSS.selected(selected))}
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
  const handleTelemChange = (telem: RemoteTelemNumericProps): void => {
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
      <RemoteTelem.Form.Numeric value={value.telem} onChange={handleTelemChange} />
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
