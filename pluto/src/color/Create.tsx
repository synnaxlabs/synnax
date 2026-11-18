// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Create.css";

import { color, group } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Theming } from "@synnaxlabs/lyra/theming";
import { uuid } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useRef,
  useState,
} from "react";

import { Access } from "@/access";
import { useAppend } from "@/color/queries";
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
  // A ref, so that two key presses in one render cannot both save.
  const savingRef = useRef(false);
  const save = (): void => {
    if (savingRef.current) return;
    const trimmed = name.trim();
    if (trimmed === "") return onCancel();
    savingRef.current = true;
    setSaving(true);
    void onSave(trimmed).then((ok) => {
      if (ok) return;
      savingRef.current = false;
      setSaving(false);
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

/** The "+" that ends a palette, and the name entry that it opens. */
const Add = (): ReactElement => {
  const { palette } = Color.usePalette();
  const { value, linked, onNamedChange } = Color.usePicker();
  const { dark } = Theming.use();
  const [naming, setNaming] = useState(false);
  const { updateAsync: append } = useAppend();
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
    const key = uuid.create();
    const created = await append({
      key,
      name,
      light: linked?.light ?? value,
      dark: linked == null ? (dark ? value : undefined) : linked.dark,
      ...(palette.key === "" ? {} : { group: palette.key }),
    });
    if (!created) return false;
    setNaming(false);
    onNamedChange(key);
    return true;
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

interface PaletteAddProps {
  palette: group.Key;
}

const PaletteAdd = ({ palette }: PaletteAddProps): ReactElement | null =>
  Access.useUpdateGranted(group.resourceID(palette)) ? <Add /> : null;

/**
 * Ends a palette of a color picker. It asks for a name in place, creates a named color
 * after the last positioned color of the palette, and links the value to it. The new
 * color copies the linked one, else the color that the picker shows in the current
 * theme. The Series palette gets none, because a copy of a series color would repeat
 * it. A palette that the user cannot update gets none, because a new color updates it.
 */
export const CreateSwatch = (): ReactElement | null => {
  const { palette } = Color.usePalette();
  if (palette.key === color.SERIES_PALETTE) return null;
  return palette.key === "" ? <Add /> : <PaletteAdd palette={palette.key} />;
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
