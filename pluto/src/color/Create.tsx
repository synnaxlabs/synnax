// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Create.css";

import { color } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { uuid } from "@synnaxlabs/x";
import { generateKeyBetween } from "fractional-indexing";
import { type KeyboardEvent, type ReactElement, type ReactNode, useState } from "react";

import { useCreate, useResultList } from "@/color/queries";
import { CSS } from "@/css";
import { Group } from "@/group";

interface NameEntryProps {
  placeholder: string;
  /** Called with the trimmed name. Resolves to whether the create succeeded. */
  onSave: (name: string) => Promise<boolean>;
  onCancel: () => void;
  children?: ReactNode;
}

/** Asks for a name in place. Enter saves, Escape cancels, and a blank name cancels. */
const NameEntry = ({
  placeholder,
  onSave,
  onCancel,
  children,
}: NameEntryProps): ReactElement => {
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const save = (): void => {
    const trimmed = name.trim();
    if (trimmed === "") return onCancel();
    setSaving(true);
    void onSave(trimmed).then((ok) => {
      if (!ok) setSaving(false);
    });
  };
  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      onCancel();
    }
  };
  return (
    <Flex.Box x align="center" gap="small" className={CSS.B("color-name-entry")}>
      {children}
      <Input.Text
        value={name}
        onChange={setName}
        placeholder={placeholder}
        aria-label={placeholder}
        autoFocus
        full="x"
        size="small"
        onKeyDown={handleKeyDown}
      />
      <Button.Button
        variant="filled"
        size="small"
        square
        tooltip="Create"
        aria-label="Create"
        status={saving ? "loading" : undefined}
        onClick={save}
      >
        <Icon.Check />
      </Button.Button>
      <Button.Button
        variant="text"
        size="small"
        square
        tooltip="Cancel"
        aria-label="Cancel"
        onClick={onCancel}
      >
        <Icon.Close />
      </Button.Button>
    </Flex.Box>
  );
};

export interface CreateSwatchProps {
  /** Called with the key of the new named color once the Core holds it. */
  onCreated: (key: color.Key) => void;
}

/**
 * Ends a palette of a color picker. It creates a named color in the palette from the
 * color that the picker shows, after asking for its name in place. The Series palette
 * gets none, because a copy of a series color would repeat it.
 */
export const CreateSwatch = ({ onCreated }: CreateSwatchProps): ReactElement | null => {
  const { palette } = Color.usePalette();
  const { value } = Color.usePicker();
  const [naming, setNaming] = useState(false);
  const { updateAsync: create } = useCreate();
  const list = useResultList({});
  if (palette.key === color.SERIES_PALETTE) return null;
  const add = (
    <Button.Button
      variant="outlined"
      square
      textColor={9}
      className={CSS.cls(CSS.B("color-create"), CSS.selected(naming))}
      tooltip="Create color"
      aria-label={`Create a color in ${palette.name}`}
      onClick={() => setNaming((n) => !n)}
    >
      <Icon.Add />
    </Button.Button>
  );
  if (!naming) return add;
  const handleSave = async (name: string): Promise<boolean> => {
    const members = (list.data ?? [])
      .filter((c) => (c.group ?? "") === palette.key && c.position != null)
      .sort(color.compare);
    const key = uuid.create();
    const created = await create({
      key,
      name,
      light: value,
      position: generateKeyBetween(members.at(-1)?.position ?? null, null),
      ...(palette.key === "" ? {} : { group: palette.key }),
    });
    if (created) onCreated(key);
    return created;
  };
  return (
    <>
      {add}
      <NameEntry
        placeholder="Color name"
        onSave={handleSave}
        onCancel={() => setNaming(false)}
      >
        <Color.Swatch value={value} size="small" aria-label="New color" />
      </NameEntry>
    </>
  );
};

/** Ends the palettes of a color picker. It creates a palette after asking its name. */
export const CreatePalette = (): ReactElement => {
  const [naming, setNaming] = useState(false);
  const { updateAsync: create } = Group.useCreate();
  if (!naming)
    return (
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        justify="start"
        className={CSS.B("color-create-palette")}
        onClick={() => setNaming(true)}
      >
        <Icon.Add />
        Create palette
      </Button.Button>
    );
  const handleSave = async (name: string): Promise<boolean> => {
    const created = await create({
      key: uuid.create(),
      name,
      scope: color.TYPE_RESOURCE_ID.type,
    });
    if (created) setNaming(false);
    return created;
  };
  return (
    <NameEntry
      placeholder="Palette name"
      onSave={handleSave}
      onCancel={() => setNaming(false)}
    />
  );
};
