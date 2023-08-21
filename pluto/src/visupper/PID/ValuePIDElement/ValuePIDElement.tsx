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

import { Color } from "@/color";
import { Telem } from "@/telem";
import { RemoteTelemNumericProps, RemoteTelem } from "@/telem/remote/main";
import { componentRenderProp } from "@/util/renderProp";
import { ValueLabeled, ValueLabeledProps } from "@/vis/value/Labeled";
import {
  PIDElementFormProps,
  StatefulPIDElementProps,
  PIDElementSpec,
} from "@/visupper/PID/PIDElement";

import { CSS, Input, Select, Space } from "@/core";

import "@/vis/PID/ValuePIDElement/ValuePIDElement.css";

export const ZERO_PROPS: ValuePIDElementProps = {
  label: "Value",
  telem: {
    channel: 0,
  },
  units: "psi",
  level: "p",
};

export interface ValuePIDElementProps extends Omit<ValueLabeledProps, "telem"> {
  telem: RemoteTelemNumericProps;
}

const ValuePIDElement = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  className,
  ...props
}: StatefulPIDElementProps<ValuePIDElementProps>): ReactElement => {
  const telem = Telem.Remote.useNumeric(pTelem);
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
      <Handle position={Position.Top} type="source" id="top" />
      <Handle position={Position.Left} type="source" id="left" />
      <Handle position={Position.Right} type="source" id="right" />
      <Handle position={Position.Bottom} type="source" id="bottom" />
    </ValueLabeled>
  );
};

const ValuePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValuePIDElementProps>): ReactElement => {
  const handleTelemChange = (telem: RemoteTelemNumericProps): void => {
    onChange({ ...value, telem });
  };

  const handleLabelChange = (label: string): void => {
    onChange({ ...value, label });
  };

  const handleUnitsChange = (units: string): void => {
    onChange({ ...value, units });
  };

  const handleDirectionChange = (direction: CrudeDirection): void => {
    onChange({ ...value, direction });
  };

  const handlecolorChange = (color: Color.Color): void => {
    onChange({ ...value, color: color.hex });
  };

  return (
    <>
      <Space direction="x" grow align="stretch">
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
      </Space>
      <Space direction="x">
        <Input.Item<Color.Crude, Color.Color, Color.SwatchProps>
          label="Color"
          value={value.color ?? Color.ZERO.setAlpha(1)}
          onChange={handlecolorChange}
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
        <RemoteTelem.Form.Numeric
          value={value.telem}
          onChange={handleTelemChange}
          grow
        />
      </Space>
    </>
  );
};

const ValuePIDElementPreview = (): ReactElement => {
  const telem = Telem.Static.useNumeric(500);
  return <ValueLabeled label="Value" units="psi" telem={telem} level="p" />;
};

export const ValuePIDElementSpec: PIDElementSpec<ValuePIDElementProps> = {
  type: "value",
  title: "Value",
  initialProps: ZERO_PROPS,
  Element: ValuePIDElement,
  Form: ValuePIDElementForm,
  Preview: ValuePIDElementPreview,
};
