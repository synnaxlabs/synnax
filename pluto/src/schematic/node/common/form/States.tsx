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
import { type ReactElement, type ReactNode, useState } from "react";

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

/** The name and entries that a new enumeration starts with. */
interface Draft {
  name: string;
  entries: enumeration.Entry[];
}

interface EnumerationFieldProps {
  /** The enumeration that the field names. */
  value?: enumeration.Key;
  onChange: (next?: enumeration.Key) => void;
  /** Returns the start of an enumeration that the user creates with no value. */
  draft: () => Draft;
  /** Renders in place of the entries with no value, given whether a create runs. */
  fallback: (creating: boolean) => ReactNode;
}

/**
 * Picks an enumeration and edits its entries. Creating an enumeration with no search
 * text asks for its name. A field that names an enumeration gets a new empty one, with
 * focus in its entries.
 */
const EnumerationField = ({
  value,
  onChange,
  draft,
  fallback,
}: EnumerationFieldProps): ReactElement => {
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const { updateAsync, variant } = Enumeration.useCreate();
  const prompt = Enumeration.useCreatePrompt();
  // The enumeration just created for a field that named one, whose entries take focus.
  const [focused, setFocused] = useState<enumeration.Key>();
  const handleChange = (next: enumeration.Key | undefined): void => {
    setFocused(undefined);
    onChange(next === NONE ? undefined : next);
  };
  const link = (created: enumeration.Key): void => {
    onChange(created);
    if (value != null) setFocused(created);
  };
  const handleCreate = (term: string): void => {
    const start: Draft = value == null ? draft() : { name: "", entries: [] };
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
      {value != null && <UsedBy enumeration={value} />}
      <Input.Item label="Entries" padHelpText={false}>
        {value == null ? (
          fallback(variant === "loading")
        ) : (
          <Enumeration.Entries
            key={value}
            enumeration={value}
            autoFocus={focused === value}
          />
        )}
      </Input.Item>
    </>
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
 * own options when it names none. An enumeration that a node with no enumeration
 * creates takes the node's options, and its name starts as the node's label.
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
            entries: get<schematic.StateMapping[]>(options).value.map(toEntry),
          })}
          // The new enumeration takes the options as they are when the create starts,
          // so they lock until it ends.
          fallback={(creating) => (
            <OwnOptions path={options} colorHidden={colorHidden} preview={creating} />
          )}
        />
      </Form.Section>
    </Form.Sections>
  );
};
