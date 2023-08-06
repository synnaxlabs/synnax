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

import { CSS, Color, ColorSwatch, CrudeColor, Input, Space, Text } from "@/core";
import { Regulator, RegulatorProps } from "@/core/vis/Regulator/Regulator";
import { componentRenderProp } from "@/util/renderProp";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/vis/PID/PIDElement";

import "@/vis/PID/RegulatorPIDElement/RegulatorPIDElement.css";

export interface RegulatorPIDElementProps extends Omit<RegulatorProps, "color"> {
  label: string;
  color: CrudeColor;
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
        {editable && <Handle position={Position.Left} type="target" />}
        {editable && <Handle position={Position.Right} type="source" />}
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
  const handleColorChange = (color: Color): void =>
    onChange({ ...value, color: color.hex });
  return (
    <Space direction="vertical" size="small">
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />
      <Input.Item<CrudeColor, Color>
        label="Color"
        value={value.color}
        onChange={handleColorChange}
      >
        {/* @ts-expect-error */}
        {componentRenderProp(ColorSwatch)}
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
