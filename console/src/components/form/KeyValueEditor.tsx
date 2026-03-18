// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/form/KeyValueEditor.css";

import { Button, Flex, Form, Icon, Input, Text } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { CSS } from "@/css";

export interface KeyValueEditorProps extends Flex.BoxProps {
  path: string;
  label: string;
  keyField?: string;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  valueType?: "string" | "number";
  valueFirst?: boolean;
}

export const KeyValueEditor: FC<KeyValueEditorProps> = ({
  path,
  label,
  keyField = "key",
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
  valueType = "string",
  valueFirst = false,
  ...rest
}) => {
  const defaultValue = valueType === "number" ? 0 : "";
  const { set } = Form.useContext();
  const value = Form.useFieldValue<Record<string, string | number>[]>(path, {
    defaultValue: [],
  });

  const entries: Record<string, string | number>[] = value;

  const setFormValue = (arr: Record<string, string | number>[]) =>
    set(path, arr.length > 0 ? arr : undefined);

  const addRow = () =>
    setFormValue([...entries, { [keyField]: "", value: defaultValue }]);

  const updateRowKey = (i: number, k: string) => {
    const updated = [...entries];
    updated[i] = { ...updated[i], [keyField]: k };
    setFormValue(updated);
  };

  const updateRowValue = (i: number, v: string | number) => {
    const updated = [...entries];
    updated[i] = { ...updated[i], value: v };
    setFormValue(updated);
  };

  const removeRow = (i: number) => setFormValue(entries.filter((_, j) => j !== i));

  return (
    <Flex.Box y gap="small" {...rest}>
      <Text.Text level="small" justify="between" size="small" color={9}>
        {label}
        <Button.Button
          onClick={addRow}
          variant="text"
          tooltip={`Add ${label.toLowerCase()}`}
          sharp
          size="small"
        >
          <Icon.Add />
        </Button.Button>
      </Text.Text>
      <Flex.Box y gap="small">
        {entries.map((entry, i) => {
          const keyInput = (
            <Input.Text
              placeholder={keyPlaceholder}
              value={(entry[keyField] as string) ?? ""}
              onChange={(v) => updateRowKey(i, v)}
            />
          );
          const valueInput =
            valueType === "number" ? (
              <Input.Numeric
                value={entry.value as number}
                onChange={(v) => updateRowValue(i, v)}
              />
            ) : (
              <Input.Text
                placeholder={valuePlaceholder}
                value={entry.value as string}
                onChange={(v) => updateRowValue(i, v)}
              />
            );
          return (
            <Flex.Box
              x
              key={i}
              align="center"
              gap="small"
              className={CSS.B("kv-row")}
            >
              {valueFirst ? valueInput : keyInput}
              {valueFirst ? keyInput : valueInput}
              <Button.Button
                variant="text"
                ghost
                size="small"
                onClick={() => removeRow(i)}
              >
                <Icon.Close />
              </Button.Button>
            </Flex.Box>
          );
        })}
      </Flex.Box>
    </Flex.Box>
  );
};
