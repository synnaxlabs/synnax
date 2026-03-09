// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Icon, Input } from "@synnaxlabs/pluto";
import { type FC, useState } from "react";

interface Entry {
  key: string;
  value: string | number;
}

export interface KeyValueEditorProps {
  path: string;
  label: string;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  valueType?: "string" | "number";
}

export const KeyValueEditor: FC<KeyValueEditorProps> = ({
  path,
  label,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
  valueType = "string",
}) => {
  const defaultValue = valueType === "number" ? 0 : "";
  const { set } = Form.useContext();
  const value = Form.useFieldValue<Record<string, string | number>>(path, {
    defaultValue: {},
  });
  const [pendingRows, setPendingRows] = useState<Entry[]>([]);

  const formEntries: Entry[] = Object.entries(value ?? {}).map(([k, v]) => ({
    key: k,
    value: v,
  }));
  const entries = [...formEntries, ...pendingRows];
  const formCount = formEntries.length;

  const syncFormValue = (record: Record<string, string | number>) =>
    set(path, Object.keys(record).length > 0 ? record : undefined);

  const addRow = () =>
    setPendingRows((prev) => [...prev, { key: "", value: defaultValue }]);

  const updateRowKey = (i: number, k: string) => {
    if (i < formCount) {
      const oldKey = formEntries[i].key;
      const next = { ...(value ?? {}) };
      delete next[oldKey];
      if (k.length > 0) next[k] = formEntries[i].value;
      syncFormValue(next);
    } else {
      const pi = i - formCount;
      const updated = [...pendingRows];
      updated[pi] = { ...updated[pi], key: k };
      if (updated[pi].key.length > 0) {
        const entry = updated[pi];
        syncFormValue({ ...(value ?? {}), [entry.key]: entry.value });
        setPendingRows(updated.filter((_, j) => j !== pi));
      } else setPendingRows(updated);
    }
  };

  const updateRowValue = (i: number, v: string | number) => {
    if (i < formCount) {
      const k = formEntries[i].key;
      syncFormValue({ ...(value ?? {}), [k]: v });
    } else {
      const pi = i - formCount;
      setPendingRows((prev) => {
        const updated = [...prev];
        updated[pi] = { ...updated[pi], value: v };
        return updated;
      });
    }
  };

  const removeRow = (i: number) => {
    if (i < formCount) {
      const next = { ...(value ?? {}) };
      delete next[formEntries[i].key];
      syncFormValue(next);
    } else setPendingRows((prev) => prev.filter((_, j) => j !== i - formCount));
  };

  return (
    <Flex.Box y gap="small">
      <Flex.Box x align="center" justify="between">
        <Input.Label>{label}</Input.Label>
        <Button.Button variant="text" size="small" onClick={addRow}>
          <Icon.Add />
        </Button.Button>
      </Flex.Box>
      {entries.map((entry, i) => (
        <Flex.Box x key={i} align="center" gap="small">
          <Input.Text
            placeholder={keyPlaceholder}
            value={entry.key}
            onChange={(v) => updateRowKey(i, v)}
          />
          {valueType === "number" ? (
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
          )}
          <Button.Button variant="text" size="small" onClick={() => removeRow(i)}>
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};
