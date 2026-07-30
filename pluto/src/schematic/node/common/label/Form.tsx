// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/form.css";

import { type direction, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Direction } from "@/direction";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Input } from "@/input";
import { Select } from "@/select";

const LABEL_INPUT_FIELD_PROPS: Partial<Input.TextProps> = { selectOnFocus: true };

interface FormProps {
  path: string;
  omit?: string[];
}

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
      className={CSS.BE("label-form", "wrap-width")}
      path={`${path}.maxInlineSize`}
      hideIfNull
      label="Label wrap width"
      inputProps={{ endContent: "px", dragScale: { x: 1, y: 0.5 } }}
      padHelpText={false}
    />
    <Base.Field<text.Level>
      hideIfNull
      visible={!omit.includes("level")}
      path={`${path}.level`}
      label="Label size"
      padHelpText={false}
    >
      {({ value, onChange }) => <Select.Text.Level value={value} onChange={onChange} />}
    </Base.Field>
    <Base.Field<Flex.Alignment>
      visible={!omit.includes("align")}
      path={`${path}.align`}
      label="Label alignment"
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
      label="Label direction"
      padHelpText={false}
      hideIfNull
    >
      {({ value, onChange }) => (
        <Direction.Select value={value} onChange={onChange} yDirection="down" />
      )}
    </Base.Field>
  </Flex.Box>
);
