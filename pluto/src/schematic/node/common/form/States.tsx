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
import { type ReactElement, type ReactNode, useState } from "react";

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

export interface EnumerationFieldProps {
  /** The enumeration that the field names. */
  value?: enumeration.Key;
  onChange: (next?: enumeration.Key) => void;
  /** Returns the start of an enumeration that the user creates with no value. */
  draft: () => Enumeration.CreatePromptParams;
  /** Shows in place of the entries when the field names no enumeration. */
  fallback?: ReactNode;
}

/**
 * Picks an enumeration and edits its entries. A new enumeration takes focus in its
 * entries. Creating one with no search text asks for its name. A field that names an
 * enumeration gets a new empty one.
 */
export const EnumerationField = ({
  value,
  onChange,
  draft,
  fallback,
}: EnumerationFieldProps): ReactElement => {
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const { updateAsync } = Enumeration.useCreate();
  const prompt = Enumeration.useCreatePrompt();
  // The enumeration just created, whose entries take focus.
  const [focused, setFocused] = useState<enumeration.Key>();
  const handleChange = (next: enumeration.Key | undefined): void => {
    setFocused(undefined);
    onChange(next === NONE ? undefined : next);
  };
  const link = (created: enumeration.Key): void => {
    onChange(created);
    setFocused(created);
  };
  const handleCreate = (term: string): void => {
    const start = value == null ? draft() : { name: "", entries: [] };
    void (async () => {
      if (term === "") {
        const created = await prompt(start);
        if (created != null) link(created);
        return;
      }
      const created = uuid.create();
      if (
        (await updateAsync({ key: created, name: term, entries: start.entries })) !==
        false
      )
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
    <>
      <Input.Item label="Enumeration" padHelpText={false}>
        <Enumeration.SelectSingle
          value={value}
          onChange={handleChange}
          allowNone
          className={CSS.cls(
            CSS.BE("states", "enumeration"),
            CSS.BE("states", "value"),
          )}
          footer={footer}
        />
      </Input.Item>
      {value == null ? (
        fallback != null && (
          <Input.Item label="Entries" padHelpText={false}>
            {fallback}
          </Input.Item>
        )
      ) : (
        <>
          <UsedBy enumeration={value} />
          <Input.Item label="Entries" padHelpText={false}>
            <Enumeration.Entries
              key={value}
              enumeration={value}
              autoFocus={focused === value}
            />
          </Input.Item>
        </>
      )}
    </>
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
  return (
    <Form.Sections x>
      <Form.Section className={CSS.B("states")}>
        <EnumerationField
          value={key}
          onChange={(next) =>
            next == null ? remove(enumerationPath) : set(enumerationPath, next)
          }
          draft={() => ({
            name: get<string>(label).value.trim() || symbolName,
            entries: [],
          })}
        />
      </Form.Section>
    </Form.Sections>
  );
};
