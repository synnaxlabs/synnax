// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type xy } from "@synnaxlabs/x";
import { Handle, Position } from "reactflow";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { type Theming } from "@/theming";
import { componentRenderProp } from "@/util/renderProp";
import { type FormProps, type Spec, type Props } from "@/vis/pid/element/element";
import { Tank, type TankProps } from "@/vis/tank/Tank";

import "@/vis/pid/element/Tank.css";

interface ElementProps extends Omit<TankProps, "telem"> {
  label: string;
}

const { Left, Right, Top, Bottom } = Position;

const Element = ({
  selected,
  editable,
  position,
  className,
  ...props
}: Props<ElementProps>): ReactElement => {
  return (
    <div className={CSS(className, CSS.B("tank-pid-element"), CSS.selected(selected))}>
      <Handle position={Left} type="source" id="a" style={{ top: "25%" }} />
      <Handle position={Left} type="source" id="c" style={{ top: "75%" }} />
      <Handle position={Right} type="source" id="e" style={{ top: "25%" }} />
      <Handle position={Right} type="source" id="g" style={{ top: "75%" }} />
      <Handle position={Top} type="source" id="j" />
      <Handle position={Bottom} type="source" id="l" />
      <Tank {...props}></Tank>
    </div>
  );
};

const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };

const Form = ({ value, onChange }: FormProps<ElementProps>): ReactElement => {
  const handleWidthChange = (width: number): void =>
    onChange({ ...value, dimensions: { ...value.dimensions, width } });
  const handleHeightChange = (height: number): void =>
    onChange({ ...value, dimensions: { ...value.dimensions, height } });
  const handleLabelChange = (label: string): void => onChange({ ...value, label });
  const handleColorChange = (color: Color.Color): void =>
    onChange({ ...value, color: color.hex });

  return (
    <>
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />

      <Align.Space direction="horizonatal">
        <Input.Item<number, number, Input.NumericProps>
          label="Width"
          value={value.dimensions.width}
          onChange={handleWidthChange}
          dragScale={DIMENSIONS_DRAG_SCALE}
        >
          {componentRenderProp(Input.Numeric)}
        </Input.Item>

        <Input.Item<number, number, Input.NumericProps>
          label="Height"
          value={value.dimensions.height}
          onChange={handleHeightChange}
          dragScale={DIMENSIONS_DRAG_SCALE}
        >
          {componentRenderProp(Input.Numeric)}
        </Input.Item>
        <Input.Item<Color.Crude, Color.Color, Color.SwatchProps>
          label="Color"
          onChange={handleColorChange}
          value={value.color}
        >
          {/* @ts-expect-error */}
          {componentRenderProp(Color.Swatch)}
        </Input.Item>
      </Align.Space>
    </>
  );
};

const Preview = ({ color }: ElementProps): ReactElement => (
  <Tank color={color} dimensions={{ width: 30, height: 40 }}></Tank>
);

const initialProps = (th: Theming.Theme): ElementProps => ({
  dimensions: { width: 100, height: 250 },
  label: "Tank",
  color: th.colors.gray.p2.hex,
});

export const TankSpec: Spec<ElementProps> = {
  type: "tank",
  title: "Tank",
  initialProps,
  Form,
  Preview,
  Element,
};
