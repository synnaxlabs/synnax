// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { CrudeDirection, Direction } from "@synnaxlabs/x";
import { Handle, Position } from "reactflow";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Select } from "@/select";
import { Remote } from "@/telem/remote";
import { Text } from "@/text";
import { componentRenderProp } from "@/util/renderProp";
import { FormProps, Spec, Props } from "@/vis/pid/element/element";
import { Valve, ValveProps } from "@/vis/valve/Valve";

import "@/vis/pid/element/Valve.css";

interface ElementProps extends Omit<ValveProps, "telem" | "color"> {
  telem: Remote.NumericSourceProps;
  label: string;
  color: Color.Crude;
}

const { Left, Top, Right, Bottom } = Position;

const Element = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  label,
  position: _,
  direction = "x",
  ...props
}: Props<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void =>
    onChange({ ...props, label, telem: pTelem });

  const parsedDirection = new Direction(direction);

  return (
    <Align.Space
      justify="center"
      align="center"
      size="small"
      className={CSS(
        CSS.B("valve-pid-element"),
        CSS.selected(selected),
        CSS.editable(editable)
      )}
      direction={parsedDirection.inverse}
    >
      <Text.Editable level="p" value={label} onChange={handleLabelChange} />
      <div className={CSS.BE("valve-pid-element", "valve-container")}>
        <Handle position={parsedDirection.isX ? Left : Top} id="a" type="source" />
        <Handle position={parsedDirection.isX ? Right : Bottom} id="b" type="source" />
        <Valve direction={direction} {...props} />
      </div>
    </Align.Space>
  );
};

const Form = ({ value, onChange }: FormProps<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });

  const handleTelemChange = (telem: Remote.NumericSourceProps): void =>
    onChange({ ...value, telem });

  const handleColorChange = (color: Color.Color): void =>
    onChange({ ...value, color: color.hex });

  const handleDirectionChange = (direction: CrudeDirection): void =>
    onChange({ ...value, direction });

  return (
    <>
      <Align.Space direction="x">
        <Input.Item<string>
          label="Label"
          value={value.label}
          onChange={handleLabelChange}
          grow
        />
        <Input.Item<Color.Crude, Color.Color, Color.SwatchProps>
          label="Color"
          onChange={handleColorChange}
          value={value.color}
        >
          {/* @ts-expect-error */}
          {componentRenderProp(ColorSwatch)}
        </Input.Item>
        <Input.Item<CrudeDirection>
          label="Direction"
          value={new Direction(value.direction ?? "x").crude}
          onChange={handleDirectionChange}
        >
          {componentRenderProp(Select.Direction)}
        </Input.Item>
      </Align.Space>
      <Remote.NumericForm value={value.telem} onChange={handleTelemChange} />
    </>
  );
};

const ValvePIDElementPreview = (): ReactElement => {
  return <Valve />;
};

const ZERO_PROPS: ElementProps = {
  label: "Valve",
  color: "#ffffff",
  telem: {
    channel: 0,
  },
};

export const ValveSpec: Spec<ElementProps> = {
  type: "valve",
  title: "Valve",
  initialProps: ZERO_PROPS,
  Element,
  Form,
  Preview: ValvePIDElementPreview,
};
