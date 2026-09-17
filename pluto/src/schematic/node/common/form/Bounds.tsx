// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Form } from "@/form";
import { type Input } from "@/input";

const STEP = 10;

export interface BoundsFieldsProps extends Pick<
  Partial<Form.NumericFieldProps>,
  "hideIfNull" | "padHelpText"
> {
  /** Path to the bounds within the symbol's config. */
  path: string;
}

/**
 * The minimum and maximum of a range, each capped by the other so the pair cannot
 * cross.
 */
export const BoundsFields = ({ path, ...rest }: BoundsFieldsProps): ReactElement => {
  const value = Form.useField<bounds.Bounds>(path, { optional: true })?.value;
  const { lower, upper } = value ?? bounds.INFINITE;
  const lowerProps = useMemo<Partial<Input.NumericProps>>(
    () => ({ step: STEP, bounds: { ...bounds.INFINITE, upper } }),
    [upper],
  );
  const upperProps = useMemo<Partial<Input.NumericProps>>(
    () => ({ step: STEP, bounds: { ...bounds.INFINITE, lower } }),
    [lower],
  );
  return (
    <>
      <Form.NumericField
        path={`${path}.lower`}
        label="Min value"
        inputProps={lowerProps}
        {...rest}
      />
      <Form.NumericField
        path={`${path}.upper`}
        label="Max value"
        inputProps={upperProps}
        {...rest}
      />
    </>
  );
};
