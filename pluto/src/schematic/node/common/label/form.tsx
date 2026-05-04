// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Direction } from "@/direction";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Input } from "@/input";
import { Select } from "@/select";
import { type Text } from "@/text";

interface FormProps {
  path: string;
  omit?: string[];
}

const MAX_INLINE_SIZE_STYLE: CSSProperties = { maxWidth: 125 };

const LABEL_INPUT_FIELD_PROPS: Partial<Input.TextProps> = { selectOnFocus: true };

export const Form = ({ path, omit = [] }: FormProps): ReactElement => (
  <Flex.Box x align="stretch">
    <Base.TextField
      path={`${path}.label`}
      label="Label"
      padHelpText={false}
      grow
      inputProps={LABEL_INPUT_FIELD_PROPS}
    />
    <Base.NumericField
      visible={!omit.includes("maxInlineSize")}
      style={MAX_INLINE_SIZE_STYLE}
      path={`${path}.maxInlineSize`}
      hideIfNull
      label="Label Wrap Width"
      inputProps={{ endContent: "px", dragScale: { x: 1, y: 0.5 } }}
      padHelpText={false}
    />
    <Base.Field<Text.Level>
      hideIfNull
      visible={!omit.includes("level")}
      path={`${path}.level`}
      label="Label Size"
      padHelpText={false}
    >
      {({ value, onChange }) => <Select.Text.Level value={value} onChange={onChange} />}
    </Base.Field>
    <Base.Field<Flex.Alignment>
      visible={!omit.includes("align")}
      path={`${path}.align`}
      label="Label Alignment"
      padHelpText={false}
      hideIfNull
    >
      {({ value, onChange }) => (
        <Select.Flex.Alignment value={value} onChange={onChange} />
      )}
    </Base.Field>
    <Base.Field<direction.Direction>
      visible={!omit.includes("direction")}
      path={`${path}.direction`}
      label="Label Direction"
      padHelpText={false}
      hideIfNull
    >
      {({ value, onChange }) => (
        <Direction.Select value={value} onChange={onChange} yDirection="down" />
      )}
    </Base.Field>
  </Flex.Box>
);
