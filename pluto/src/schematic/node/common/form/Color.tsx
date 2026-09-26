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

import { Color } from "@/color";
import { Form } from "@/form";

export interface ColorFieldProps extends Omit<
  Form.FieldProps<color.Crude | undefined>,
  "placeholder"
> {
  /** The color the symbol renders with while the field is unset. */
  placeholder?: color.Crude;
}

export const ColorField = ({
  placeholder,
  ...props
}: ColorFieldProps): ReactElement => (
  <Form.Field label="Color" align="start" padHelpText={false} {...props}>
    {({ value, onChange, ...rest }) => (
      <Color.Swatch
        value={value}
        onChange={onChange}
        onClear={() => onChange(undefined)}
        placeholder={placeholder}
        {...rest}
        bordered
      />
    )}
  </Form.Field>
);
