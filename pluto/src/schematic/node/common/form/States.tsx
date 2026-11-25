// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/States.css";

import { enumeration, schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type deep, uuid } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Access } from "@/access";
import { CSS } from "@/css";
import { Enumeration } from "@/enumeration";

const NONE = "none";

interface UsedByProps {
  enumeration: enumeration.Key;
}

/** Names how many schematics use the enumeration, this one included. */
const UsedBy = ({ enumeration: key }: UsedByProps): ReactElement | null => {
  const res = Enumeration.useResult({ key, includeUses: true });
  const n = (res.data?.uses ?? []).filter(
    ({ type }) => type === schematic.TYPE_RESOURCE_ID.type,
  ).length;
  if (n === 0) return null;
  return (
    <Input.Item label="Used by" padHelpText={false}>
      <Text.Text level="p" className={CSS.BE("states", "value")}>
        <Icon.Schematic />
        {n} {n === 1 ? "schematic" : "schematics"}
      </Text.Text>
    </Input.Item>
  );
};

export interface StatesProps {
  /** Path to the enumeration that the node names. */
  enumeration: deep.Path<enumeration.Key | undefined>;
  /** Path to the node's label, which names an enumeration that the node creates. */
  label: deep.Path<string>;
  /** Names an enumeration that the node creates when it has no search text or label. */
  symbolName: string;
}

/**
 * Picks the enumeration that a node draws and edits its entries. A new enumeration
 * starts empty, with focus in its entries. Creating one with no search text asks for
 * its name, which starts as the node's label when the node names no enumeration.
 */
export const States = ({
  enumeration: enumerationPath,
  label,
  symbolName,
}: StatesProps): ReactElement => {
  const { set, remove, get } = Form.useContext();
  const key = Form.useFieldValue(enumerationPath, { optional: true }) ?? undefined;
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const { updateAsync } = Enumeration.useCreate();
  const prompt = Enumeration.useCreatePrompt();
  // The enumeration just created, whose entries take focus.
  const [focused, setFocused] = useState<enumeration.Key>();
  const handleChange = (next: enumeration.Key | undefined): void => {
    setFocused(undefined);
    if (next == null || next === NONE) remove(enumerationPath);
    else set(enumerationPath, next);
  };
  const link = (created: enumeration.Key): void => {
    set(enumerationPath, created);
    setFocused(created);
  };
  const handleCreate = (term: string): void => {
    void (async () => {
      if (term === "") {
        const name = key == null ? get<string>(label).value.trim() || symbolName : "";
        const created = await prompt({ name, entries: [] });
        if (created != null) link(created);
        return;
      }
      const created = uuid.create();
      if ((await updateAsync({ key: created, name: term, entries: [] })) !== false)
        link(created);
    })();
  };
  const footer = (
    <>
      <Menu.Divider />
      {canCreate && (
        <Select.CreateItem<enumeration.Key, enumeration.Enumeration>
          resourceName="enumeration"
          onCreate={handleCreate}
        />
      )}
      <Select.Item<enumeration.Key> itemKey={NONE} textValue="No enumeration">
        <Icon.None />
        No enumeration
      </Select.Item>
    </>
  );
  return (
    <Form.Sections x>
      <Form.Section className={CSS.B("states")}>
        <Input.Item label="Enumeration" padHelpText={false}>
          <Enumeration.SelectSingle
            value={key}
            onChange={handleChange}
            allowNone
            className={CSS.cls(
              CSS.BE("states", "enumeration"),
              CSS.BE("states", "value"),
            )}
            footer={footer}
          />
        </Input.Item>
        {key != null && (
          <>
            <UsedBy enumeration={key} />
            <Input.Item label="Entries" padHelpText={false}>
              <Enumeration.Entries
                key={key}
                enumeration={key}
                autoFocus={focused === key}
              />
            </Input.Item>
          </>
        )}
      </Form.Section>
    </Form.Sections>
  );
};
