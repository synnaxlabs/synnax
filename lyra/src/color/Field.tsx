// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Input, type InputProps } from "@/color/Input";
import { Form } from "@/form";
import { Input as BaseInput } from "@/input";

export interface FieldProps
  extends
    Omit<BaseInput.ItemProps, "children" | "onChange">,
    Pick<InputProps, "size" | "onlyChangeOnBlur"> {
  /** Dot-separated path into the form values. */
  path: string;
  /** The color the theme paints while the field is absent. */
  fallback: color.Crude;
}

/**
 * A form row that edits an optional color. While the field is absent, the input shows
 * Auto; clearing the input or picking Auto clears the field.
 */
export const Field = ({
  path,
  fallback,
  size,
  onlyChangeOnBlur,
  label = "Color",
  align = "start",
  padHelpText = false,
  ...rest
}: FieldProps): ReactElement => {
  const value = Form.useFieldValue<color.Crude>(path, { optional: true });
  const { set } = Form.useContext();
  return (
    <BaseInput.Item label={label} align={align} padHelpText={padHelpText} {...rest}>
      <Input
        value={value ?? undefined}
        fallback={fallback}
        onChange={(c) => set(path, c)}
        size={size}
        onlyChangeOnBlur={onlyChangeOnBlur}
      />
    </BaseInput.Item>
  );
};
