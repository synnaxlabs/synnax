// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/States.css";

import {
  type color as colorRef,
  enumeration,
  type schematic,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { color, type deep, uuid } from "@synnaxlabs/x";
import { type ReactElement } from "react";

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

interface OptionsProps {
  path: deep.Path<schematic.StateMapping[]>;
  colorDisabled: boolean;
}

const Options = ({ path, colorDisabled }: OptionsProps): ReactElement => {
  const { set } = Form.useContext();
  const options = Form.useFieldValue<schematic.StateMapping[]>(path);
  const resolve = Color.useResolve();
  const entries = options.map(toEntry);
  const write = (next: enumeration.Entry[]): void =>
    set(
      path,
      next.map((e) => toOption(e, resolve)),
    );
  return (
    <Enumeration.Table
      value={entries}
      abnormalDisabled
      colorDisabled={colorDisabled}
      onSet={(entry) => write(entries.map((e) => (e.key === entry.key ? entry : e)))}
      onAdd={(added) => write([...entries, ...added])}
      onRemove={(i) => write(entries.filter((_, j) => j !== i))}
      onMove={(from, to) => {
        const next = [...entries];
        const [moved] = next.splice(from, 1);
        next.splice(to, 0, moved);
        write(next);
      }}
    />
  );
};

interface CreateItemProps {
  /** Called with the search text when the user creates an enumeration. */
  onCreate: (term: string) => void;
}

const CreateItem = ({ onCreate }: CreateItemProps): ReactElement => {
  const { close } = Dialog.useContext();
  const term = Select.useTerm();
  return (
    <Button.CreateListItem
      size="small"
      onClick={() => {
        close();
        onCreate(term);
      }}
    >
      Create enumeration
    </Button.CreateListItem>
  );
};

interface UsedByProps {
  enumeration: enumeration.Key;
}

/** Names how many schematics use the enumeration, when another one does. */
const UsedBy = ({ enumeration: key }: UsedByProps): ReactElement | null => {
  const res = Enumeration.useResultDependents({ key });
  const n = res.data?.length ?? 0;
  if (n < 2) return null;
  return (
    <Input.Item label="Used by" padHelpText={false}>
      <Text.Text level="p">
        <Icon.Schematic />
        {n} schematics
      </Text.Text>
    </Input.Item>
  );
};

export interface StatesProps {
  /** Path to the enumeration that the node names. */
  enumeration: deep.Path<enumeration.Key | undefined>;
  /** Path to the options that the node draws when it names no enumeration. */
  options: deep.Path<schematic.StateMapping[]>;
  /** Path to the node's label, which names an enumeration that the node creates. */
  label: deep.Path<string>;
  /** Names an enumeration that the node creates when it has no search text or label. */
  symbolName: string;
  /** Shows the Color cell of each option empty, for a node that draws no color. */
  colorDisabled?: boolean;
}

/**
 * Picks the enumeration that a node draws and edits its entries, or edits the node's
 * own options when it names none. Creating an enumeration moves the options into it.
 */
export const States = ({
  enumeration: enumerationPath,
  options,
  label,
  symbolName,
  colorDisabled = false,
}: StatesProps): ReactElement => {
  const { set, remove, get } = Form.useContext();
  const key = Form.useFieldValue(enumerationPath, { optional: true }) ?? undefined;
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const { updateAsync } = Enumeration.useCreate();
  const handleChange = (next: enumeration.Key | undefined): void => {
    if (next == null || next === NONE) remove(enumerationPath);
    else set(enumerationPath, next);
  };
  const handleCreate = (term: string): void => {
    const name =
      [term, get(label).value].map((n) => n.trim()).find((n) => n !== "") ?? symbolName;
    const created = uuid.create();
    const entries = get<schematic.StateMapping[]>(options).value.map(toEntry);
    void (async () => {
      const ok = await updateAsync({ key: created, name, entries });
      if (ok !== false) set(enumerationPath, created);
    })();
  };
  const footer = (
    <>
      <Menu.Divider />
      <Select.Item<enumeration.Key> itemKey={NONE} textValue="No enumeration">
        <Icon.None />
        No enumeration
      </Select.Item>
      {canCreate && <CreateItem onCreate={handleCreate} />}
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
            className={CSS.BE("states", "enumeration")}
            footer={footer}
          />
        </Input.Item>
        {key != null && <UsedBy enumeration={key} />}
        <Input.Item label="Entries" padHelpText={false}>
          {key == null ? (
            <Options path={options} colorDisabled={colorDisabled} />
          ) : (
            <Enumeration.Entries enumeration={key} />
          )}
        </Input.Item>
      </Form.Section>
    </Form.Sections>
  );
};
