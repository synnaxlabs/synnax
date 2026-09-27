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

import { Swatch, type SwatchProps } from "@/color/Swatch";
import { Form } from "@/form";
import { Input } from "@/input";

export interface FieldProps
  extends
    Omit<Input.ItemProps, "children" | "onChange">,
    Pick<SwatchProps, "size" | "onlyChangeOnBlur"> {
  /** Dot-separated path into the form values. */
  path: string;
  /** The color the theme paints while the field is absent. */
  fallback: color.Crude;
}

/**
 * A form row that edits an optional color. The swatch shows the fallback, marked as
 * auto, while the field is absent, and the picker's Auto swatch clears the field.
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
    <Input.Item label={label} align={align} padHelpText={padHelpText} {...rest}>
      <Swatch
        value={value ?? undefined}
        fallback={fallback}
        onChange={(c) => set(path, c)}
        size={size}
        onlyChangeOnBlur={onlyChangeOnBlur}
        bordered
      />
    </Input.Item>
  );
};
