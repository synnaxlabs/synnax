// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/States.css";

import { type color as colorRef, enumeration, schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { color, type deep, uuid } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Access } from "@/access";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Enumeration } from "@/enumeration";

const NONE = "none";

const toEntry = (o: schematic.StateMapping): enumeration.Entry => ({
  key: o.key,
  value: o.value,
  name: o.name,
  abnormal: false,
  ...(o.color != null &&
    !color.equals(o.color, color.TRANSPARENT) && {
      color: { type: "fixed", value: o.color },
    }),
});

// An option holds a fixed color, so a named pick keeps the color it has now.
const toOption = (
  e: enumeration.Entry,
  resolve: (ref: colorRef.Ref) => color.Color | undefined,
): schematic.StateMapping => {
  const c = e.color == null ? undefined : resolve(e.color);
  return { key: e.key, value: e.value, name: e.name, ...(c != null && { color: c }) };
};

interface OwnOptionsProps {
  path: deep.Path<schematic.StateMapping[]>;
  colorHidden: boolean;
  preview: boolean;
}

const OwnOptions = ({ path, colorHidden, preview }: OwnOptionsProps): ReactElement => {
  const { set, get } = Form.useContext();
  const options = Form.useFieldValue<schematic.StateMapping[]>(path);
  const resolve = Color.useResolve();
  const write = (
    change: (current: enumeration.Entry[]) => enumeration.Entry[],
  ): void => {
    const current = get<schematic.StateMapping[]>(path).value.map(toEntry);
    set(
      path,
      change(current).map((e) => toOption(e, resolve)),
    );
  };
  return (
    <Enumeration.Table
      value={options.map(toEntry)}
      preview={preview}
      abnormalHidden
      colorHidden={colorHidden}
      onChange={({ changed, added, removed }) =>
        write((current) => [
          ...current
            .filter((e) => !removed.includes(e.key))
            .map((e) => changed.find((c) => c.key === e.key) ?? e),
          ...added,
        ])
      }
      onMove={(from, to) =>
        write((current) => {
          const next = [...current];
          const [moved] = next.splice(from, 1);
          next.splice(to, 0, moved);
          return next;
        })
      }
    />
  );
};

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

export interface StatesProps extends Pick<Enumeration.TableProps, "colorHidden"> {
  /** Path to the enumeration that the node names. */
  enumeration: deep.Path<enumeration.Key | undefined>;
  /** Path to the options that the node draws when it names no enumeration. */
  options: deep.Path<schematic.StateMapping[]>;
  /** Path to the node's label, which names an enumeration that the node creates. */
  label: deep.Path<string>;
  /** Names an enumeration that the node creates when it has no search text or label. */
  symbolName: string;
}

/**
 * Picks the enumeration that a node draws and edits its entries, or edits the node's
 * own options when it names none. Creating an enumeration with no search text asks for
 * its name, which starts as the node's label when the options move into it. A node that
 * names an enumeration gets a new empty one, with focus in its entries.
 */
export const States = ({
  enumeration: enumerationPath,
  options,
  label,
  symbolName,
  colorHidden = false,
}: StatesProps): ReactElement => {
  const { set, remove, get } = Form.useContext();
  const key = Form.useFieldValue(enumerationPath, { optional: true }) ?? undefined;
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const { updateAsync, variant } = Enumeration.useCreate();
  const prompt = Enumeration.useCreatePrompt();
  // The enumeration just created for a node that named one, whose entries take focus.
  const [focused, setFocused] = useState<enumeration.Key>();
  const handleChange = (next: enumeration.Key | undefined): void => {
    setFocused(undefined);
    if (next == null || next === NONE) remove(enumerationPath);
    else set(enumerationPath, next);
  };
  const link = (created: enumeration.Key): void => {
    set(enumerationPath, created);
    if (key != null) setFocused(created);
  };
  const handleCreate = (term: string): void => {
    // A node that names an enumeration hides its own options, so none move.
    const entries =
      key == null ? get<schematic.StateMapping[]>(options).value.map(toEntry) : [];
    void (async () => {
      if (term === "") {
        const name = key == null ? get<string>(label).value.trim() || symbolName : "";
        const created = await prompt({ name, entries });
        if (created != null) link(created);
        return;
      }
      const created = uuid.create();
      if ((await updateAsync({ key: created, name: term, entries })) !== false)
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
        {key != null && <UsedBy enumeration={key} />}
        <Input.Item label="Entries" padHelpText={false}>
          {key == null ? (
            // The new enumeration takes the options as they are when the create
            // starts, so they lock until it ends.
            <OwnOptions
              path={options}
              colorHidden={colorHidden}
              preview={variant === "loading"}
            />
          ) : (
            <Enumeration.Entries
              key={key}
              enumeration={key}
              autoFocus={focused === key}
            />
          )}
        </Input.Item>
      </Form.Section>
    </Form.Sections>
  );
};
