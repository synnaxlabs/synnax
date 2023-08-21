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
import { CSS, ColorSwatch, SwatchProps, Input, Select, Space, Text } from "@/core";
import { Valve, ValveProps } from "@/vis/valve/Valve";
import { RemoteTelem, RemoteTelemNumericProps } from "@/telem/remote/main";
import { componentRenderProp } from "@/util/renderProp";
import {
  PIDElementFormProps,
  PIDElementSpec,
  StatefulPIDElementProps,
} from "@/visupper/PID/PIDElement";

import "@/vis/PID/ValvePIDElement/ValvePIDElement.css";

export interface ValvePIDElementProps extends Omit<ValveProps, "telem" | "color"> {
  telem: RemoteTelemNumericProps;
  label: string;
  color: Color.Crude;
}

const ValvePIDElement = ({
  selected,
  editable,
  telem: pTelem,
  onChange,
  label,
  position: _,
  direction = "x",
  ...props
}: StatefulPIDElementProps<ValvePIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void =>
    onChange({ ...props, label, telem: pTelem });

  const parsedDirection = new Direction(direction);

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
      direction={parsedDirection.inverse}
    >
      <Text.Editable level="p" value={label} onChange={handleLabelChange} />
      <div className={CSS.BE("valve-pid-element", "valve-container")}>
        <Handle
          position={parsedDirection.isX ? Position.Left : Position.Top}
          id="a"
          type="source"
        />
        <Handle
          position={parsedDirection.isX ? Position.Right : Position.Bottom}
          id="b"
          type="source"
        />
        <Valve direction={direction} {...props} />
      </div>
    </Space>
  );
};

const ValvePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValvePIDElementProps>): ReactElement => {
  const handleLabelChange = (label: string): void => onChange({ ...value, label });

  const handleTelemChange = (telem: RemoteTelemNumericProps): void =>
    onChange({ ...value, telem });

  const handleColorChange = (color: Color.Color): void =>
    onChange({ ...value, color: color.hex });

  const handleDirectionChange = (direction: CrudeDirection): void => {
    onChange({ ...value, direction });
  };

  return (
    <>
      <Space direction="x">
        <Input.Item<string>
          label="Label"
          value={value.label}
          onChange={handleLabelChange}
          grow
        />
        <Input.Item<Color.Crude, Color.Color, SwatchProps>
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
      </Space>
      <RemoteTelem.Form.Numeric value={value.telem} onChange={handleTelemChange} />
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
