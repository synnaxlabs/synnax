// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Input, Text } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { useEffect, useMemo } from "react";

export type Entry<K extends string, V extends string | number> = {
  [k in K]: string;
} & { value: V };

export interface KeyValueEditorProps<K extends string, V extends string | number>
  extends Flex.BoxProps {
  path: string;
  label: string;
  keyField: K;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  valueType?: V extends number ? "number" : "string";
  valueFirst?: boolean;
}

export const KeyValueEditor = <K extends string, V extends string | number>({
  path,
  label,
  keyField,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
  valueType,
  valueFirst = false,
  ...rest
}: KeyValueEditorProps<K, V>): React.ReactElement => {
  const vt = valueType ?? "string";
  const { set, mode } = Form.useContext();
  const preview = mode === "preview" ? true : undefined;
  const value = Form.useFieldValue<Entry<K, V>[]>(path, { defaultValue: [] });

  useEffect(() => {
    // weird stuff we have to do to deal with migrations where the previous value is an
    // object, and the task schema configuration was not applied because the task was
    // set in Flux via a list retrieve that did not run schema.config.parse on the task.
    // This means the first time the form is rendered with values from retrieveList, we
    // get the v0 object instead of the array. So we have to do this jank thing where we
    // set the value to an empty array as the previous values do not work.
    // https://linear.app/synnax/issue/SY-3943/strongly-type-tasks-and-devices-in-flux
    if (!Array.isArray(value)) set(path, []);
  }, []);
  const entries = Array.isArray(value) ? value : [];

  // Both directions and the column order read off these, so valueFirst cannot leave a
  // cell bound to the other field.
  const keyIndex = valueFirst ? 1 : 0;
  const valueIndex = valueFirst ? 0 : 1;

  const rows = useMemo(
    () =>
      entries.map((entry) => {
        const row: Input.TableCell[] = [];
        row[keyIndex] = entry[keyField];
        row[valueIndex] = entry.value;
        return row;
      }),
    [entries, keyField, keyIndex, valueIndex],
  );

  const handleRowsChange = (next: Input.TableCell[][]) =>
    set(
      path,
      next.length > 0
        ? next.map(
            (row) =>
              ({ [keyField]: row[keyIndex], value: row[valueIndex] }) as Entry<K, V>,
          )
        : undefined,
    );

  const keyColumn = (
    <Input.TableColumn key="key" name={caseconv.capitalize(keyField)} type="string">
      {(p) => <Input.Text {...p} placeholder={keyPlaceholder} />}
    </Input.TableColumn>
  );
  const valueColumn =
    vt === "number" ? (
      <Input.TableColumn key="value" name="Value" />
    ) : (
      <Input.TableColumn key="value" name="Value" type="string">
        {(p) => <Input.Text {...p} placeholder={valuePlaceholder} />}
      </Input.TableColumn>
    );

  const columns = [];
  columns[keyIndex] = keyColumn;
  columns[valueIndex] = valueColumn;

  return (
    <Flex.Box y gap="small" {...rest}>
      <Text.Text level="small" size="small" color={9}>
        {label}
      </Text.Text>
      <Input.Table value={rows} onChange={handleRowsChange} preview={preview}>
        {columns}
      </Input.Table>
    </Flex.Box>
  );
};
