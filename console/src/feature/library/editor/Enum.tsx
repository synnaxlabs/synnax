// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { useNestedStatus } from "@/feature/library/editor/useNestedStatus";

/** Appends the value after the largest one, so a new row starts out unique. */
const createRow = (rows: Input.TableCell[][]): Input.TableCell[] => {
  const values = rows.map(([value]) => Number(value));
  return [values.length === 0 ? 0 : Math.max(...values) + 1, ""];
};

export interface EnumProps {
  /** The form path of the enum entry. */
  path: string;
}

/** Edits the values of an enum as a grid of value and name. */
export const Enum = ({ path }: EnumProps): ReactElement => {
  const valuesPath = `${path}.values`;
  const { set } = Form.useContext();
  const values = Form.useFieldValue<library.EnumValue[]>(valuesPath);
  const rows = useMemo(() => values.map(({ value, name }) => [value, name]), [values]);
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      valuesPath,
      next.map(([value, name]) => ({ value: Number(value), name: String(name) })),
    );
  const status = useNestedStatus(valuesPath);
  const index = deep.getIndex(status?.key?.split(".")[3] ?? "");
  let helpText = status?.message;
  if (index != null) helpText = `Value ${index + 1}: ${helpText}`;
  return (
    <Input.Item label="Values" padHelpText helpText={helpText} status={status?.variant}>
      <Input.Table
        value={rows}
        onChange={handleChange}
        createRow={createRow}
        rowLabel={(i) => (i + 1).toString()}
      >
        <Input.TableColumn name="Value" />
        <Input.TableColumn name="Name" type="string" />
      </Input.Table>
    </Input.Item>
  );
};
