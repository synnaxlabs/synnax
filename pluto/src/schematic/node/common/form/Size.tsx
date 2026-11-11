// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { LEVEL_SIZES, SIZE_LEVELS } from "@/schematic/node/common/size";

const path = deep.path<{ size: Component.Size; level: text.Level }>();

export const SizeField = (
  props: Partial<Form.FieldProps<Component.Size>>,
): ReactElement => (
  <Form.Field path={path.size} label="Size" padHelpText={false} {...props}>
    {selectSize}
  </Form.Field>
);

const selectSize = Component.renderProp(Component.SelectSize);

/**
 * A size field for symbols that store a text level beside their size. Picking a size
 * also sets the level, so the text matches the other symbols at that size.
 */
export const ScaledSizeField = (
  props: Partial<Form.FieldProps<Component.Size>>,
): ReactElement => (
  <Form.Field
    path={path.size}
    label="Size"
    padHelpText={false}
    onChange={(size, { set }) => set(path.level, SIZE_LEVELS[size])}
    {...props}
  >
    {selectScaledSize}
  </Form.Field>
);

// A click on the selected size reports null. It must still apply the level, because a
// saved symbol's level can differ from the one its size gives.
const SelectScaledSize = ({
  value,
  onChange,
}: {
  value: Component.Size;
  onChange: (size: Component.Size) => void;
}): ReactElement => (
  <Component.SelectSize
    allowNone
    value={value}
    onChange={(size: Component.Size | null) => onChange(size ?? value)}
  />
);

const selectScaledSize = Component.renderProp(SelectScaledSize);

/** A size field for symbols that store a text level instead of a size. */
export const LevelSizeField = (
  props: Partial<Form.FieldProps<text.Level>>,
): ReactElement => (
  <Form.Field path={path.level} label="Size" padHelpText={false} {...props}>
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
