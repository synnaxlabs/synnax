// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

export interface BackendFieldProps<B extends string> {
  /** The name of every backend the device schema allows. */
  names: Record<B, string>;
  /** The backends a user may choose, in order. */
  offered: readonly B[];
}

/** Stands in for an absent backend, which Form.Field would otherwise hide. */
const NONE = "";

/**
 * Selects the backend at properties.backend from the offered backends. A device that
 * already names a backend outside them still shows it as an option.
 */
export const BackendField = <B extends string>({
  names,
  offered,
}: BackendFieldProps<B>): ReactElement => (
  <Form.Field<B | typeof NONE>
    path="properties.backend"
    label="Backend"
    defaultValue={NONE}
  >
    {({ value, onChange, ...rest }) => (
      <Select.Simple<B>
        {...rest}
        resourceName="backend"
        value={value === NONE ? undefined : value}
        onChange={(next: B | null) => {
          if (next != null) onChange(next);
        }}
      >
        {(value === NONE || offered.includes(value)
          ? offered
          : [...offered, value]
        ).map((key) => (
          <Select.Item<B> key={key} itemKey={key}>
            {names[key]}
          </Select.Item>
        ))}
      </Select.Simple>
    )}
  </Form.Field>
);
