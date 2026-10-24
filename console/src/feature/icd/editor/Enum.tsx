// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { useNestedStatus } from "@/feature/icd/editor/useNestedStatus";
import { CSS } from "@/platform/css";

/** Appends the value after the largest one, so a new row starts out unique. */
const createRow = (rows: Input.TableCell[][]): Input.TableCell[] => {
  const values = rows.map(([value]) => Number(value));
  return [values.length === 0 ? 0 : Math.max(...values) + 1, ""];
};

export interface EnumProps {
  /** The form path of the enum entry. */
  path: string;
}

/** Edits an enum: its name, and its values as a grid of value and name. */
export const Enum = ({ path }: EnumProps): ReactElement => {
  const valuesPath = `${path}.values`;
  const { set } = Form.useContext();
  const values = Form.useFieldValue<icd.EnumValue[]>(valuesPath);
  const rows = useMemo(() => values.map(({ value, name }) => [value, name]), [values]);
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      valuesPath,
      next.map(([value, name]) => ({ value: Number(value), name: String(name) })),
    );
  const status = useNestedStatus(valuesPath);
  const valueKey = status?.key?.slice(valuesPath.length + 1) ?? "";
  const index = deep.getIndex(deep.element(valueKey, 0));
  let helpText = status?.message;
  if (index != null) helpText = `Value ${index + 1}: ${helpText}`;
  return (
    <Form.Sections className={CSS.BE("icd-editor", "details")}>
      <Form.Section title="Enum">
        <Form.TextField path={`${path}.name`} label="Name" />
      </Form.Section>
      <Form.Section title="Values">
        <Flex.Box y gap="small">
          <Input.Table
            value={rows}
            onChange={handleChange}
            createRow={createRow}
            rowLabel={(i) => (i + 1).toString()}
          >
            <Input.TableColumn name="Value" width={16} />
            <Input.TableColumn name="Name" type="string" />
          </Input.Table>
          {helpText != null && (
            <Input.HelpText variant={status?.variant}>{helpText}</Input.HelpText>
          )}
        </Flex.Box>
      </Form.Section>
    </Form.Sections>
  );
};
