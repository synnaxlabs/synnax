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

import {
  CSS,
  Color,
  ColorSwatch,
  ColorSwatchProps,
  CrudeColor,
  Input,
  Space,
  Text,
} from "@/core";
import { Valve, ValveProps } from "@/core/vis/Valve/Valve";
import { RangeNumerictelemProps } from "@/telem/range/aether";
import { RangeNumericTelemForm } from "@/telem/range/forms";
import { componentRenderProp } from "@/util/renderProp";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/vis/PID/PIDElement";

import "@/vis/PID/ValvePIDElement/ValvePIDElement.css";

export interface ValvePIDElementProps extends Omit<ValveProps, "telem" | "color"> {
  telem: RangeNumerictelemProps;
  label: string;
  color: CrudeColor;
}

const ValvePIDElement = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  label,
  position: _,
  ...props
}: StatefulPIDElementProps<ValvePIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void =>
    onChange({ ...props, label, telem: pTelem });
  return (
    <Space
      justify="center"
      align="center"
      size="small"
      className={CSS(
        CSS.B("valve-pid-element"),
        CSS.selected(selected),
        CSS.editable(editable)
      )}
    >
      <Text.Editable level="p" value={label} onChange={handleLabelChange} />
      <div className={CSS.BE("valve-pid-element", "valve-container")}>
        {editable && <Handle position={Position.Left} type="target" />}
        {editable && <Handle position={Position.Right} type="source" />}
        <Valve {...props} />
      </div>
    </Space>
  );
};

const ValvePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValvePIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });

  const handleTelemChange = (telem: RangeNumerictelemProps): void =>
    onChange({ ...value, telem });

  const handleColorChange = (color: Color): void =>
    onChange({ ...value, color: color.hex });

  return (
    <>
      <Space direction="x">
        <Input.Item<string>
          label="Label"
          value={value.label}
          onChange={handleLabelChange}
        />
        <Input.Item<CrudeColor, Color, ColorSwatchProps>
          label="Color"
          onChange={handleColorChange}
          value={value.color}
        >
          {/* @ts-expect-error */}
          {componentRenderProp(ColorSwatch)}
        </Input.Item>
      </Space>
      <RangeNumericTelemForm value={value.telem} onChange={handleTelemChange} />
    </>
  );
};

const ValvePIDElementPreview = (): ReactElement => {
  return <Valve />;
};

const ZERO_PROPS: ValvePIDElementProps = {
  label: "Valve",
  color: "#ffffff",
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
