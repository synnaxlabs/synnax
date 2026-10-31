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
import { type CSSProperties, type ReactElement } from "react";

import { parseAddress, toModicon } from "@/feature/modbus/task/address";
import {
  addressSpan,
  type ReadChannel,
  type WriteChannel,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";

export interface AddressFieldProps extends Pick<Input.TextProps, "variant"> {
  /** The path of the channel in the form. */
  path: string;
  /** Marks the addresses red, for a channel that shares one with another channel. */
  overlapping?: boolean;
}

/**
 * The address of a channel. It takes a zero-based offset or a six-digit Modicon address
 * such as 400001, and shows the Modicon form after the offset. A channel that spans
 * several addresses shows its last one after the first. Text that does not parse keeps
 * the stored address and marks the field until a valid entry.
 */
export const AddressField = ({
  path,
  variant,
  overlapping = false,
}: AddressFieldProps): ReactElement => {
  const channel = Form.useFieldValue<ReadChannel | WriteChannel>(path);
  const field = Form.useField<number>(`${path}.address`);
  const invalid = field.status.variant !== "success";
  const [lo, hi] = addressSpan(channel);
  return (
    <Input.Item
      showLabel={false}
      showHelpText={false}
      status={field.status.variant}
      helpText={field.status.message}
    >
      <div
        className={CSS.cls(
          CSS.BE("modbus", "address"),
          overlapping && CSS.BEM("modbus", "address", "overlap"),
        )}
        style={{ "--console-modbus-digits": String(lo).length } as CSSProperties}
      >
        <Input.Text
          value={field.value.toString()}
          onChange={(raw) => {
            const parsed = parseAddress(channel.type, raw);
            if ("error" in parsed)
              return field.setStatus({ variant: "error", message: parsed.error });
            // The status clears on its own only when the value changes.
            if (invalid) field.setStatus({ variant: "success", message: "" });
            field.onChange(parsed.address);
          }}
          onlyChangeOnBlur
          selectOnFocus
          preview={field.preview}
          variant={variant}
          endContent={toModicon(channel.type, lo)}
          aria-label="Address"
        />
        {hi > lo && <span className={CSS.BE("modbus", "address-end")}>{`–${hi}`}</span>}
      </div>
    </Input.Item>
  );
};
