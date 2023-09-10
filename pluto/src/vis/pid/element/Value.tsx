// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Handle, Position } from "reactflow";

import {direction} from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Select } from "@/select";
import { Remote } from "@/telem/remote";
import { Static } from "@/telem/static";
import { type Theming } from "@/theming";
import { componentRenderProp } from "@/util/renderProp";
import { type FormProps, type Props, type Spec } from "@/vis/pid/element/element";
import { ValueLabeled, type ValueLabeledProps } from "@/vis/value/Labeled";

import "@/vis/pid/element/Value.css";

interface ElementProps extends Omit<ValueLabeledProps, "telem"> {
  telem: Remote.NumericSourceProps;
}

const { Top, Left, Right, Bottom } = Position;

const Element = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  className,
  ...props
}: Props<ElementProps>): ReactElement => {
  const telem = Remote.useNumericSource(pTelem);
  const onLabelChange = (label: string): void => {
    onChange({ ...props, label, telem: pTelem });
  };

  return (
    <ValueLabeled
      className={CSS(className, CSS.B("value-pid-element"), CSS.selected(selected))}
      {...props}
      telem={telem}
      onLabelChange={onLabelChange}
    >
      <Handle position={Top} type="source" id="top" />
      <Handle position={Left} type="source" id="left" />
      <Handle position={Right} type="source" id="right" />
      <Handle position={Bottom} type="source" id="bottom" />
    </ValueLabeled>
  );
};

const Form = ({ value, onChange }: FormProps<ElementProps>): ReactElement => {
  const handleTelemChange = (telem: Remote.NumericSourceProps): void => {
    onChange({ ...value, telem });
  };

  const handleLabelChange = (label: string): void => {
    onChange({ ...value, label });
  };

  const handleUnitsChange = (units: string): void => {
    onChange({ ...value, units });
  };

  const handleDirectionChange = (direction: direction.Direction): void => {
    onChange({ ...value, direction });
  };

  const handlecolorChange = (color: Color.Color): void => {
    onChange({ ...value, color: color.hex });
  };

  return (
    <>
      <Align.Space direction="x" grow align="stretch">
        <Input.Item<string>
          label="Label"
          value={value.label}
          onChange={handleLabelChange}
          grow
        />
        <Input.Item<string>
          label="Units"
          value={value.units}
          onChange={handleUnitsChange}
          grow
        />
      </Align.Space>
      <Align.Space direction="x">
        <Input.Item<Color.Crude, Color.Color, Color.SwatchProps>
          label="Color"
          value={value.color ?? Color.ZERO.setAlpha(1)}
          onChange={handlecolorChange}
        >
          {/* @ts-expect-error */}
          {componentRenderProp(Color.Swatch)}
        </Input.Item>
        <Input.Item<direction.Direction>
          label="Direction"
          value={value.direction ?? "x"}
          onChange={handleDirectionChange}
        >
          {componentRenderProp(Select.Direction)}
        </Input.Item>
        <Remote.NumericSourceForm
          value={value.telem}
          onChange={handleTelemChange}
          grow
        />
      </Align.Space>
    </>
  );
};

const Preview = ({ color }: ElementProps): ReactElement => {
  const telem = Static.useNumeric(500);
  return (
    <ValueLabeled label="Value" units="psi" telem={telem} level="p" color={color} />
  );
};

export const initialProps = (th: Theming.Theme): ElementProps => ({
  label: "Value",
  color: th.colors.gray.p2.hex,
  telem: {
    channel: 0,
  },
  units: "psi",
  level: "p",
});

export const ValueSpec: Spec<ElementProps> = {
  type: "value",
  title: "Value",
  initialProps,
  Element,
  Form,
  Preview,
};
