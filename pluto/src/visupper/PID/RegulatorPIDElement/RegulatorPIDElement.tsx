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

import { Color } from "@/color";
import { componentRenderProp } from "@/util/renderProp";
import { Regulator, RegulatorProps } from "@/vis/regulator/Regulator";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/visupper/PID/PIDElement";

import { CSS, Input, Space, Text } from "@/core";

import "@/vis/PID/RegulatorPIDElement/RegulatorPIDElement.css";

export interface RegulatorPIDElementProps extends Omit<RegulatorProps, "color"> {
  label: string;
  color: Color.Crude;
}

const RegulatorPIDElement = ({
  selected,
  editable,
  onChange,
  label,
  ...props
}: StatefulPIDElementProps<RegulatorPIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...props, label });

  return (
    <Space
      justify="center"
      align="center"
      size="small"
      className={CSS(CSS.B("regulator-pid-element"), CSS.selected(selected))}
    >
      <Text.Editable level="p" value={label} onChange={handleLabelChange} />
      <div>
        <Handle id="a" position={Position.Left} type="source" style={{ top: "75%" }} />
        <Handle id="b" position={Position.Right} type="source" style={{ top: "75%" }} />
        <Regulator {...props} />
      </div>
    </Space>
  );
};

const RegulatorPIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<RegulatorPIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });
  const handleColorChange = (color: Color.Color): void =>
    onChange({ ...value, color: color.hex });
  return (
    <Space direction="vertical" size="small">
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />
      <Input.Item<Color.Crude, Color.Color>
        label="Color"
        value={value.color}
        onChange={handleColorChange}
      >
        {/* @ts-expect-error */}
        {componentRenderProp(Color.Swatch)}
      </Input.Item>
    </Space>
  );
};

const RegulatorPIDElementPreview = (): ReactElement => {
  return <Regulator width="50" />;
};

const ZERO_PROPS: RegulatorPIDElementProps = {
  label: "Regulator",
  color: "#000000",
};

export const RegulatorPIDElementSpec: PIDElementSpec<RegulatorPIDElementProps> = {
  type: "regulator",
  title: "Regulator",
  initialProps: ZERO_PROPS,
  Element: RegulatorPIDElement,
  Form: RegulatorPIDElementForm,
  Preview: RegulatorPIDElementPreview,
};
