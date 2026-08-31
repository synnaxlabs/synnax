// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { Form } from "@/form";
import { LEVEL_SIZES, SIZE_LEVELS } from "@/schematic/node/common/size";

export const SizeField = (
  props: Partial<Form.FieldProps<Component.Size>>,
): ReactElement => (
  <Form.Field<Component.Size>
    path="size"
    label="Size"
    hideIfNull
    padHelpText={false}
    {...props}
  >
    {selectSize}
  </Form.Field>
);

const selectSize = Component.renderProp(Component.SelectSize);

/** A size field for symbols that store a text level instead of a size. */
export const LevelSizeField = (
  props: Partial<Form.FieldProps<text.Level>>,
): ReactElement => (
  <Form.Field<text.Level>
    path="level"
    label="Size"
    hideIfNull
    padHelpText={false}
    {...props}
  >
    {selectLevelSize}
  </Form.Field>
);

const SelectLevelSize = ({
  value,
  onChange,
}: {
  value: text.Level;
  onChange: (level: text.Level) => void;
}): ReactElement => (
  <Component.SelectSize
    value={LEVEL_SIZES[value]}
    onChange={(size: Component.Size) => onChange(SIZE_LEVELS[size])}
  />
);

const selectLevelSize = Component.renderProp(SelectLevelSize);
