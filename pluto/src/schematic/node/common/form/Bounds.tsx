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
import type { closedBounds, deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

const STEP = 10;

const MESSAGE = "Min value must be less than max value";

type Edge = keyof closedBounds.ClosedBounds;

export interface BoundsFieldsProps extends Pick<
  Partial<Form.NumericFieldProps>,
  "padHelpText"
> {
  /** Path to the bounds within the symbol's config. */
  path: deep.Path<closedBounds.ClosedBounds>;
}

/**
 * The minimum and maximum of a range. The Core refuses a minimum that is not below the
 * maximum, so the fields refuse such an edit with an error and keep the stored range.
 */
export const BoundsFields = ({ path, ...rest }: BoundsFieldsProps): ReactElement => {
  const ctx = Form.useContext();
  const field = (edge: Edge, label: string): ReactElement => (
    <Form.Field<number> path={path[edge]} label={label} {...rest}>
      {({ value, onChange }) => (
        <Input.Numeric
          value={value}
          step={STEP}
          onChange={(next) => {
            const range = { ...ctx.get(path).value, [edge]: next };
            if (range.lower >= range.upper) {
              ctx.setStatus(path[edge], { variant: "error", message: MESSAGE });
              return;
            }
            ctx.clearStatus(path.lower);
            ctx.clearStatus(path.upper);
            onChange(next);
          }}
        />
      )}
    </Form.Field>
  );
  return (
    <>
      {field("lower", "Min value")}
      {field("upper", "Max value")}
    </>
  );
};
