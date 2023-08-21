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

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Text } from "@/text";
import { componentRenderProp } from "@/util/renderProp";
import { FormProps, Spec, Props } from "@/vis/pid/element/element";
import { Regulator } from "@/vis/regulator";

interface ElementProps extends Omit<Regulator.RegulatorProps, "color"> {
  label: string;
  color: Color.Crude;
}

const Element = ({
  selected,
  editable,
  onChange,
  label,
  ...props
}: Props<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...props, label });

  return (
    <Align.Space
      justify="center"
      align="center"
      size="small"
      className={CSS(CSS.B("regulator-pid-element"), CSS.selected(selected))}
    >
      <Text.Editable level="p" value={label} onChange={handleLabelChange} />
      <div>
        <Handle id="a" position={Position.Left} type="source" style={{ top: "75%" }} />
        <Handle id="b" position={Position.Right} type="source" style={{ top: "75%" }} />
        <Regulator.Regulator {...props} />
      </div>
    </Align.Space>
  );
};

const Form = ({ value, onChange }: FormProps<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });
  const handleColorChange = (color: Color.Color): void =>
    onChange({ ...value, color: color.hex });
  return (
    <Align.Space direction="vertical" size="small">
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
    </Align.Space>
  );
};

const Preview = (): ReactElement => {
  return <Regulator.Regulator width="50" />;
};

const ZERO_PROPS: ElementProps = {
  label: "Regulator",
  color: "#000000",
};

export const RegulatorSpec: Spec<ElementProps> = {
  type: "regulator",
  title: "Regulator",
  initialProps: ZERO_PROPS,
  Element,
  Form,
  Preview,
};
