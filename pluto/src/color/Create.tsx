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
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Theming } from "@synnaxlabs/lyra/theming";
import { uuid } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

import { Access } from "@/access";
import { NO_PALETTE } from "@/color/library";
import { useCreate, useCreatePalette } from "@/color/queries";
import { CSS } from "@/css";

interface NameEntryProps {
  placeholder: string;
  /** Called with the trimmed name. */
  onSave: (name: string) => Promise<void>;
  onCancel: () => void;
  /** If true, the create is saving, and nothing cancels it. */
  saving: boolean;
  children?: ReactNode;
}

/** Asks for a name in place. Enter saves, Escape cancels, and a blank name cancels. */
const NameEntry = ({
  placeholder,
  onSave,
  onCancel,
  saving,
  children,
}: NameEntryProps): ReactElement => {
  const [name, setName] = useState("");
  const save = (): void => {
    if (saving) return;
    const trimmed = name.trim();
    if (trimmed === "") return onCancel();
    void onSave(trimmed);
  };
  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (!saving) onCancel();
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
        disabled={saving}
        onClick={onCancel}
      >
        <Icon.Close />
      </Button.Button>
    </Flex.Box>
  );
};

/**
 * The "+" that ends a palette, and the name entry that it opens. A create moves the
 * focus to the new swatch, and a cancel moves it back to the "+".
 */
const Add = (): ReactElement => {
  const { palette, focused } = Color.usePalette();
  const { value, linked, link } = Color.usePicker();
  const { dark } = Theming.use();
  const [naming, setNaming] = useState(false);
  const { updateAsync: create, variant } = useCreate();
  const saving = variant === "loading";
  const plus = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (focused) plus.current?.focus();
  }, [focused]);
  const cancel = (): void => {
    plus.current?.focus();
    setNaming(false);
  };
  const add = (
    <Button.Button
      ref={plus}
      variant="text"
      square
      textColor={9}
      className={CSS.cls(CSS.selected(naming))}
      tooltip="Create color"
      aria-label={`Create a color in ${palette.name}`}
      disabled={saving}
      onClick={() => setNaming((n) => !n)}
    >
      <Icon.Add />
    </Button.Button>
  );
  if (!naming) return add;
  const handleSave = async (name: string): Promise<void> => {
    const key = uuid.create();
    const created = await create({
      key,
      name,
      light: linked?.light ?? value,
      dark: linked == null ? (dark ? value : undefined) : linked.dark,
      ...(palette.key === NO_PALETTE ? {} : { group: palette.key }),
    });
    if (!created) return;
    setNaming(false);
    link(key);
  };
  return (
    <>
      {add}
      <NameEntry
        placeholder="Color name"
        onSave={handleSave}
        onCancel={cancel}
        saving={saving}
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
 * Ends a palette with a "+" that creates a named color at its end and links the value
 * to it. The Series palette and palettes that the user cannot update get none.
 */
export const CreateSwatch = (): ReactElement | null => {
  const { palette } = Color.usePalette();
  if (palette.key === color.SERIES_PALETTE) return null;
  return palette.key === NO_PALETTE ? <Add /> : <PaletteAdd palette={palette.key} />;
};

/**
 * Ends the palettes of a color picker. It creates a palette after asking its name, and
 * clears the search so that the palette shows. A create moves the focus to the "+" of
 * the new palette. A cancel, or a create for a user who cannot add to the palette,
 * moves it back to the button that opened the entry.
 */
export const CreatePalette = (): ReactElement => {
  const { clearSearch, focusPalette } = Color.usePicker();
  // No policy can name the key of a new palette, so only a grant on every palette gives
  // it a "+".
  const extendable = Access.useUpdateGranted(group.TYPE_RESOURCE_ID);
  const [naming, setNaming] = useState(false);
  const [focused, setFocused] = useState(false);
  const { updateAsync: create, variant } = useCreatePalette();
  const handleSave = async (name: string): Promise<void> => {
    const key = uuid.create();
    const created = await create({ key, name });
    if (!created) return;
    setNaming(false);
    clearSearch();
    if (extendable) focusPalette(key);
    else setFocused(true);
  };
  return (
    <Flex.Box y empty>
      <Divider.Divider x padded className={CSS.BE("color-create-palette", "divider")} />
      {naming ? (
        <NameEntry
          placeholder="Palette name"
          onSave={handleSave}
          saving={variant === "loading"}
          onCancel={() => {
            setFocused(true);
            setNaming(false);
          }}
        />
      ) : (
        <Button.CreateListItem
          size="small"
          className={CSS.B("color-create-palette")}
          autoFocus={focused}
          onClick={() => {
            setFocused(false);
            setNaming(true);
          }}
        >
          Create palette
        </Button.CreateListItem>
      )}
    </Flex.Box>
  );
};
