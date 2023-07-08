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
import { Valve, ValveProps } from "@/core/vis/Valve/Valve";
import { RangeNumerictelemProps } from "@/telem/range/aether";
import { RangeNumericTelemForm } from "@/telem/range/forms";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/vis/pid/PIDElement";

import "@/vis/pid/ValvePIDElement/ValvePIDElement.css";

export interface ValvePIDElementProps extends Omit<ValveProps, "telem"> {
  telem: RangeNumerictelemProps;
  label: string;
}

const ValvePIDElement = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  className,
  ...props
}: StatefulPIDElementProps<ValvePIDElementProps>): ReactElement => {
  return (
    <>
      {editable && <Handle position={Position.Left} type="target" />}
      {editable && <Handle position={Position.Right} type="source" />}
      <Valve
        className={CSS(className, CSS.B("valve-pid-element"), CSS.selected(selected))}
        {...props}
      />
    </>
  );
};

const ValvePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValvePIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => {
    onChange({ ...value, label });
  };

  const handleTelemChange = (telem: RangeNumerictelemProps): void => {
    onChange({ ...value, telem });
  };

  return (
    <>
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />
      <RangeNumericTelemForm value={value.telem} onChange={handleTelemChange} />
    </>
  );
};

const ValvePIDElementPreview = (): ReactElement => {
  return <Valve />;
};

const ZERO_PROPS: ValvePIDElementProps = {
  label: "Valve",
  telem: {
    channel: 0,
  },
};

export const ValvePIDElementSpec: PIDElementSpec<ValvePIDElementProps> = {
  type: "valve",
  title: "Valve",
  initialProps: ZERO_PROPS,
  Element: ValvePIDElement,
  Form: ValvePIDElementForm,
  Preview: ValvePIDElementPreview,
};
