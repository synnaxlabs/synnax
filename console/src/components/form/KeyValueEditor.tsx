// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Icon, Input } from "@synnaxlabs/pluto";
import { type FC, useCallback, useState } from "react";

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
  const value = Form.useFieldValue<Record<string, string>>(path, {
    optional: true,
  });
  const entries: Entry[] = Object.entries(value ?? {}).map(([k, v]) => ({
    key: k,
    value: v,
  }));
  const [draft, setDraft] = useState<Entry[]>(entries);

  const sync = useCallback(
    (next: Entry[]) => {
      setDraft(next);
      const record: Record<string, string> = {};
      for (const { key, value: v } of next) if (key.length > 0) record[key] = v;
      set(path, Object.keys(record).length > 0 ? record : undefined);
    },
    [set, path],
  );

  const addRow = useCallback(
    () => sync([...draft, { key: "", value: "" }]),
    [draft, sync],
  );

  const updateRow = useCallback(
    (i: number, field: "key" | "value", v: string) => {
      const next = [...draft];
      next[i] = { ...next[i], [field]: v };
      sync(next);
    },
    [draft, sync],
  );

  const removeRow = useCallback(
    (i: number) => sync(draft.filter((_, j) => j !== i)),
    [draft, sync],
  );

  return (
    <Flex.Box y gap="small">
      <Flex.Box x align="center" justify="between">
        <Input.Label>{label}</Input.Label>
        <Button.Button variant="text" size="small" onClick={addRow}>
          <Icon.Add />
        </Button.Button>
      </Flex.Box>
      {draft.map((entry, i) => (
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
