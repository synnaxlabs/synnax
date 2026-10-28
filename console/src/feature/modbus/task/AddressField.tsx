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
import { type ReactElement, useState } from "react";

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
 * The address of a channel. It takes a zero-based offset or a six-digit Modicon
 * address such as 400001, and shows the Modicon form after the offset. Text that does
 * not parse keeps the stored address and marks the field until the next edit.
 */
export const AddressField = ({ path }: AddressFieldProps): ReactElement => {
  const type = Form.useFieldValue<ChannelType>(`${path}.type`);
  const [error, setError] = useState<string | null>(null);
  return (
    <Form.Field<number> path={`${path}.address`} showLabel={false} showHelpText={false}>
      {({ value, onChange, preview }) => (
        <Input.Text
          value={value.toString()}
          onChange={(raw) => {
            const parsed = parseAddress(type, raw);
            if ("error" in parsed) return setError(parsed.error);
            setError(null);
            onChange(parsed.address);
          }}
          onlyChangeOnBlur
          selectOnFocus
          preview={preview}
          status={error == null ? undefined : "error"}
          tooltip={error ?? ""}
          hideTooltip={error == null}
          endContent={toModicon(type, value)}
          aria-label="Address"
        />
      )}
    </Form.Field>
  );
};
