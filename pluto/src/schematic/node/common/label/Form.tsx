// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Direction } from "@/direction";
import { type Config, WRAP_WIDTH_BOUNDS } from "@/schematic/node/common/label/Label";

const LABEL_INPUT_FIELD_PROPS: Partial<Input.TextProps> = { selectOnFocus: true };
const MULTILINE_INPUT_FIELD_PROPS: Partial<Input.TextProps> = {
  ...LABEL_INPUT_FIELD_PROPS,
  area: true,
};

interface FormProps {
  path: deep.Path<Config>;
  omit?: string[];
  /** Makes the label field a text area, so a user can enter newlines. */
  multiline?: boolean;
}

const WRAP_WIDTH_DRAG_SCALE = { x: 1, y: 0.5 };

const WrapWidthField = ({ path }: { path: deep.Target<number> }): ReactElement => {
  const value = Base.useFieldValue(path);
  const { set } = Base.useContext();
  return (
    <Input.Item label="Wrap width" padHelpText={false}>
      <Input.Numeric
        value={value}
        onChange={(v) => set(path, v)}
        endContent="px"
        bounds={WRAP_WIDTH_BOUNDS}
        integer
        dragScale={WRAP_WIDTH_DRAG_SCALE}
      />
    </Input.Item>
  );
};

export const Form = ({ path, omit = [], multiline }: FormProps): ReactElement => (
  <Flex.Box x align="stretch">
    <Base.TextField
      path={path.label}
      label="Label"
      padHelpText={false}
      grow
      inputProps={multiline ? MULTILINE_INPUT_FIELD_PROPS : LABEL_INPUT_FIELD_PROPS}
    />
    {!omit.includes("maxInlineSize") && <WrapWidthField path={path.maxInlineSize} />}
    <Base.Field
      visible={!omit.includes("level")}
      path={path.level}
      label="Size"
      padHelpText={false}
    >
      {({ value, onChange }) => <Select.Text.Level value={value} onChange={onChange} />}
    </Base.Field>
    <Base.Field
      visible={!omit.includes("align")}
      path={path.align}
      label="Alignment"
      padHelpText={false}
    >
      {({ value, onChange }) => (
        <Select.Flex.Alignment value={value} onChange={onChange} />
      )}
    </Base.Field>
    <Base.Field
      visible={!omit.includes("direction")}
      path={path.direction}
      label="Direction"
      padHelpText={false}
    >
      {({ value, onChange }) => (
        <Direction.Select value={value} onChange={onChange} yDirection="down" />
      )}
    </Base.Field>
  </Flex.Box>
);
