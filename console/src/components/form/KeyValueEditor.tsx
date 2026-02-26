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
  value: string;
}

export interface KeyValueEditorProps {
  path: string;
  label: string;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
}

export const KeyValueEditor: FC<KeyValueEditorProps> = ({
  path,
  label,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
}) => {
  const { set } = Form.useContext();
  const value = Form.useFieldValue<Record<string, string>>(path, { defaultValue: {} });
  const [pendingRows, setPendingRows] = useState<Entry[]>([]);

  const formEntries: Entry[] = Object.entries(value ?? {}).map(([k, v]) => ({
    key: k,
    value: v,
  }));
  const entries = [...formEntries, ...pendingRows];
  const formCount = formEntries.length;

  const syncFormValue = (record: Record<string, string>) =>
    set(path, Object.keys(record).length > 0 ? record : undefined);

  const addRow = () => setPendingRows((prev) => [...prev, { key: "", value: "" }]);

  const updateRow = (i: number, field: "key" | "value", v: string) => {
    if (i < formCount) {
      const oldKey = formEntries[i].key;
      const next = { ...(value ?? {}) };
      if (field === "key") {
        delete next[oldKey];
        if (v.length > 0) next[v] = formEntries[i].value;
      } else next[oldKey] = v;

      syncFormValue(next);
    } else {
      const pi = i - formCount;
      const updated = [...pendingRows];
      updated[pi] = { ...updated[pi], [field]: v };
      if (updated[pi].key.length > 0) {
        const entry = updated[pi];
        syncFormValue({ ...(value ?? {}), [entry.key]: entry.value });
        setPendingRows(updated.filter((_, j) => j !== pi));
      } else setPendingRows(updated);
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
            onChange={(v) => updateRow(i, "key", v)}
          />
          <Input.Text
            placeholder={valuePlaceholder}
            value={entry.value}
            onChange={(v) => updateRow(i, "value", v)}
          />
          <Button.Button variant="text" size="small" onClick={() => removeRow(i)}>
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};
