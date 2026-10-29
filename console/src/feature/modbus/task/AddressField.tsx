// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

import {
  type ChannelType,
  parseAddress,
  toModicon,
} from "@/feature/modbus/task/address";

export interface AddressFieldProps {
  /** The path of the channel in the form. */
  path: string;
}

/**
 * The address of a channel. It takes a zero-based offset or a six-digit Modicon address
 * such as 400001, and shows the Modicon form after the offset. Text that does not parse
 * keeps the stored address and marks the field until a valid entry.
 */
export const AddressField = ({ path }: AddressFieldProps): ReactElement => {
  const type = Form.useFieldValue<ChannelType>(`${path}.type`);
  const field = Form.useField<number>(`${path}.address`);
  const invalid = field.status.variant !== "success";
  return (
    <Input.Text
      value={field.value.toString()}
      onChange={(raw) => {
        const parsed = parseAddress(type, raw);
        if ("error" in parsed)
          return field.setStatus({ variant: "error", message: parsed.error });
        // The status clears on its own only when the value changes.
        if (invalid) field.setStatus({ variant: "success", message: "" });
        field.onChange(parsed.address);
      }}
      onlyChangeOnBlur
      selectOnFocus
      preview={field.preview}
      status={invalid ? field.status.variant : undefined}
      tooltip={field.status.message}
      hideTooltip={!invalid}
      endContent={toModicon(type, field.value)}
      aria-label="Address"
    />
  );
};
