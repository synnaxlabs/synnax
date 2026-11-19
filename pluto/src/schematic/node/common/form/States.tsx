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
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { color, type deep, uuid } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useState } from "react";

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
  preview: boolean;
}

const Options = ({ path, colorDisabled, preview }: OptionsProps): ReactElement => {
  const { set, get } = Form.useContext();
  const options = Form.useFieldValue<schematic.StateMapping[]>(path);
  const resolve = Color.useResolve();
  // One table change can call several handlers, so each changes the last write.
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
      abnormalDisabled
      colorDisabled={colorDisabled}
      onSet={(entry) =>
        write((current) => current.map((e) => (e.key === entry.key ? entry : e)))
      }
      onAdd={(added) => write((current) => [...current, ...added])}
      onRemove={(i) => write((current) => current.filter((_, j) => j !== i))}
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
  const res = Enumeration.useResultDependents({ key });
  const n = (res.data ?? []).filter(
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

interface CreateModalProps {
  onCreate: (name: string) => void;
  onClose: () => void;
  loading: boolean;
}

/** Asks for the name of a new enumeration. */
const CreateModal = ({
  onCreate,
  onClose,
  loading,
}: CreateModalProps): ReactElement => {
  const [name, setName] = useState("");
  const trimmed = name.trim();
  const handleCreate = (): void => {
    if (trimmed !== "" && !loading) onCreate(trimmed);
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "Enter") handleCreate();
  };
  return (
    <Dialog.Frame
      variant="modal"
      visible
      onVisibleChange={(visible) => !visible && onClose()}
      background={0}
    >
      <Modal.Frame>
        <Modal.Header icon={<Icon.Enumeration />}>Enumeration.Create</Modal.Header>
        <Modal.Body>
          <Input.Text
            value={name}
            onChange={setName}
            onKeyDown={handleKeyDown}
            placeholder="Enumeration name"
            aria-label="Enumeration name"
            variant="text"
            autoFocus
            level="h3"
            full="x"
          />
        </Modal.Body>
        <Modal.Footer>
          <Nav.Bar.End>
            <Button.Button
              variant="filled"
              status={loading ? "loading" : undefined}
              disabled={trimmed === ""}
              onClick={handleCreate}
            >
              Create
            </Button.Button>
          </Nav.Bar.End>
        </Modal.Footer>
      </Modal.Frame>
    </Dialog.Frame>
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
 * own options when it names none. Creating an enumeration moves the options into it. A
 * node that names one gets a new empty enumeration, with focus in its entries, and asks
 * for its name when the search is empty.
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
  const { updateAsync, variant } = Enumeration.useCreate();
  // The enumeration just created for a node that named one, whose entries take focus.
  const [focused, setFocused] = useState<enumeration.Key>();
  const [naming, setNaming] = useState(false);
  const handleChange = (next: enumeration.Key | undefined): void => {
    if (next == null || next === NONE) remove(enumerationPath);
    else set(enumerationPath, next);
  };
  const create = (name: string): void => {
    const created = uuid.create();
    // A node that names an enumeration hides its own options, so none move.
    const entries =
      key == null ? get<schematic.StateMapping[]>(options).value.map(toEntry) : [];
    void (async () => {
      if ((await updateAsync({ key: created, name, entries })) === false) return;
      set(enumerationPath, created);
      setNaming(false);
      if (key != null) setFocused(created);
    })();
  };
  const handleCreate = (term: string): void => {
    if (term === "" && key != null) return setNaming(true);
    create(
      [term, get(label).value].map((n) => n.trim()).find((n) => n !== "") ?? symbolName,
    );
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
            <Options
              path={options}
              colorDisabled={colorDisabled}
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
      {naming && (
        <CreateModal
          onCreate={create}
          onClose={() => setNaming(false)}
          loading={variant === "loading"}
        />
      )}
    </Form.Sections>
  );
};
