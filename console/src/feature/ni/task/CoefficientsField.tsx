// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Input } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

export interface CoefficientsFieldProps {
  path: string;
  label: string;
}

/**
 * Edits the coefficients of a polynomial, ordered from the constant term upwards, so
 * that index i holds the coefficient of x^i.
 * @param path - The form path of the coefficient array.
 * @param label - The heading shown above the rows.
 */
export const CoefficientsField = ({
  path,
  label,
}: CoefficientsFieldProps): ReactElement => {
  const { value, onChange, preview, status } = Form.useField<number[]>(path);
  const rows = useMemo(() => value.map((coeff) => [coeff]), [value]);
  return (
    <Input.Item
      label={label}
      helpText={status.message}
      status={status.variant}
      padHelpText
    >
      <Input.Table
        value={rows}
        onChange={(next) => onChange(next.map(([coeff]) => Number(coeff)))}
        rowLabel={(index) => `c${index}`}
        preview={preview}
      >
        <Input.TableColumn />
      </Input.Table>
    </Input.Item>
  );
};
