// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Bounds, CrudeDirection, Direction } from "@synnaxlabs/x";
import { Handle, Position } from "reactflow";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Select } from "@/select";
import { Bool } from "@/telem/bool";
import { Control } from "@/telem/control";
import { Remote } from "@/telem/remote";
import { Text } from "@/text";
import { componentRenderProp } from "@/util/renderProp";
import { FormProps, Spec, Props } from "@/vis/pid/element/element";
import { Valve, ValveProps } from "@/vis/valve/Valve";

import "@/vis/pid/element/Valve.css";

interface ElementProps extends Omit<ValveProps, "telem" | "color" | "source" | "sink"> {
  source: Remote.NumericSourceProps;
  sink: Control.NumericSinkProps;
  label: string;
  color: Color.Crude;
}

const { Left, Top, Right, Bottom } = Position;

const Element = ({
  selected,
  editable,
  onChange,
  label,
  position: _,
  direction = "x",
  source,
  sink,
  ...props
}: Props<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void =>
    onChange({ ...props, label, source, sink });

  const parsedDirection = new Direction(direction);

  const sourceN = Remote.useNumericSource(source);
  const sinkN = Control.useNumeric(sink);
  const sourceB = Bool.useNumericConverterSource({
    wrap: sourceN,
    trueBound: new Bounds(0.9, 1.1),
  });
  const sinkB = Bool.useNumericConverterSink({
    wrap: sinkN,
    truthy: 1,
    falsy: 0,
  });

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
        <Valve source={sourceB} sink={sinkB} direction={direction} {...props} />
      </div>
    </Align.Space>
  );
};

const Form = ({ value, onChange }: FormProps<ElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });

  const handleSourceChange = (source: Remote.NumericSourceProps): void =>
    onChange({ ...value, source });

  const handleSinkChange = (sink: Control.NumericSinkProps): void =>
    onChange({ ...value, sink });

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
          {componentRenderProp(Color.Swatch)}
        </Input.Item>
        <Input.Item<CrudeDirection>
          label="Direction"
          value={new Direction(value.direction ?? "x").crude}
          onChange={handleDirectionChange}
        >
          {componentRenderProp(Select.Direction)}
        </Input.Item>
      </Align.Space>
      <Align.Space direction="x">
        <Remote.NumericSourceForm
          label="Input Channel"
          value={value.source}
          onChange={handleSourceChange}
          grow
        />
        <Control.NumericSinkForm
          label="Output Channel"
          value={value.sink}
          onChange={handleSinkChange}
          grow
        />
      </Align.Space>
    </>
  );
};

const ValvePIDElementPreview = (): ReactElement => {
  return <Valve />;
};

const ZERO_PROPS: ElementProps = {
  label: "Valve",
  color: "#ffffff",
  source: {
    channel: 0,
  },
  sink: {
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
