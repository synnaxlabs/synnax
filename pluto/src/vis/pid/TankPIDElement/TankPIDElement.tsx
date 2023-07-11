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
  ColorT,
  Input,
  InputNumberProps,
  Space,
} from "@/core";
import { Tank, TankProps } from "@/core/vis/Tank/Tank";
import { componentRenderProp } from "@/util/renderProp";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/vis/pid/PIDElement";

import "@/vis/pid/TankPIDElement/TankPIDElement.css";

import { CrudeXY } from "@synnaxlabs/x";

export interface TankPIDElementProps extends Omit<TankProps, "telem"> {
  label: string;
}

const { Left, Right, Top, Bottom } = Position;

const TankPIDElement = ({
  selected,
  editable,
  ...props
}: StatefulPIDElementProps<TankPIDElementProps>): ReactElement => {
  return (
    <>
      {editable && (
        <>
          <Handle position={Left} type="target" id="a" style={{ top: "25%" }} />
          <Handle position={Left} type="source" id="a" style={{ top: "22%" }} />
          <Handle position={Left} type="target" id="b" style={{ top: "75%" }} />
          <Handle position={Left} type="source" id="b" style={{ top: "78%" }} />
          <Handle position={Right} type="target" id="c" style={{ top: "25%" }} />
          <Handle position={Right} type="source" id="d" style={{ top: "22%" }} />
          <Handle position={Right} type="target" id="d" style={{ top: "75%" }} />
          <Handle position={Right} type="source" id="d" style={{ top: "78%" }} />
          <Handle position={Top} type="target" id="e" />
          <Handle position={Bottom} type="target" id="e" />
        </>
      )}
      <Tank className={CSS(CSS.selected(selected))} {...props}></Tank>
    </>
  );
};

const DIMENSIONS_DRAG_SCALE: CrudeXY = { y: 2, x: 0.25 };

const TankPIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<TankPIDElementProps>): ReactElement => {
  const handleWidthChange = (width: number): void =>
    onChange({ ...value, dimensions: { ...value.dimensions, width } });
  const handleHeightChange = (height: number): void =>
    onChange({ ...value, dimensions: { ...value.dimensions, height } });
  const handleLabelChange = (label: string): void => onChange({ ...value, label });
  const handleColorChange = (color: ColorT): void => onChange({ ...value, color });

  return (
    <>
      <Input.Item<string>
        label="Label"
        value={value.label}
        onChange={handleLabelChange}
      />

      <Space direction="horizonatal">
        <Input.Item<number, number, InputNumberProps>
          label="Width"
          value={value.dimensions.width}
          onChange={handleWidthChange}
          dragScale={DIMENSIONS_DRAG_SCALE}
        >
          {componentRenderProp(Input.Number)}
        </Input.Item>

        <Input.Item<number, number, InputNumberProps>
          label="Height"
          value={value.dimensions.height}
          onChange={handleHeightChange}
          dragScale={DIMENSIONS_DRAG_SCALE}
        >
          {componentRenderProp(Input.Number)}
        </Input.Item>
      </Space>
      <Input.Item<ColorT, Color, ColorSwatchProps>
        label="Color"
        onChange={handleColorChange}
        value={value.color}
      >
        {/* @ts-expect-error */}
        {componentRenderProp(ColorSwatch)}
      </Input.Item>
    </>
  );
};

const TankPIDElementPreview = (): ReactElement => {
  return <Tank color={ZERO_PROPS.color} dimensions={{ width: 100, height: 50 }}></Tank>;
};

const ZERO_PROPS = {
  dimensions: { width: 100, height: 250 },
  label: "Tank",
  color: "#ffffff",
};

export const TankPIDElementSpec: PIDElementSpec<TankPIDElementProps> = {
  type: "tank",
  title: "Tank",
  initialProps: ZERO_PROPS,
  Element: TankPIDElement,
  Form: TankPIDElementForm,
  Preview: TankPIDElementPreview,
};
