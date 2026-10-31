// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

export interface NumericFieldProps extends Omit<Base.NumericFieldProps, "children"> {
  /** The value the symbol uses while the field is absent. */
  fallback: number;
}

/** Edits an optional number, showing the fallback while the field is absent. */
export const NumericField = ({
  fallback,
  inputProps,
  ...rest
}: NumericFieldProps): ReactElement => (
  <Base.Field<number | undefined, number> {...rest}>
    {({ value, ...p }) => (
      <Input.Numeric {...p} {...inputProps} value={value ?? fallback} />
    )}
  </Base.Field>
);
