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
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color as xcolor, uuid } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { Access } from "@/access";
import { NO_PALETTE, usePalettes } from "@/color/library";
import { createFormSchema, useCreate, useCreatePalette } from "@/color/queries";
import { Fields } from "@/color/View";

interface NameEntryProps {
  placeholder: string;
  /** Called with the trimmed name. */
  onSave: (name: string) => Promise<void>;
  onCancel: () => void;
  /** If true, the create is saving, and nothing cancels it. */
  saving: boolean;
}

/** Asks for a name in place. Enter saves, Escape cancels, and a blank name cancels. */
const NameEntry = ({
  placeholder,
  onSave,
  onCancel,
  saving,
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
    <Flex.Box x align="center" gap="small">
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

interface CreateViewProps {
  /**
   * The palette that the new color starts in. Defaults to the first palette that the
   * user can add to.
   */
  palette?: group.Key;
}

/**
 * Creates a named color from a view of the picker, and links the value to it. It starts
 * with the color that the picker shows. In the light theme, Dark starts on Auto, so an
 * edit of Light carries to it.
 */
const CreateView = ({ palette: initialPalette }: CreateViewProps): ReactElement => {
  const { value, linked, link } = Color.usePicker();
  const { back } = Dialog.useViews();
  const { dark } = Theming.use();
  const all = usePalettes().filter(
    ({ key }) => key !== color.SERIES_PALETTE && key !== NO_PALETTE,
  );
  const updatable = Access.useUpdatable(all.map(({ key }) => group.resourceID(key)));
  const palettes = all.filter(({ key }) => updatable.some((id) => id.key === key));
  const [picked, setPicked] = useState(initialPalette);
  const palette = picked ?? palettes[0]?.key ?? NO_PALETTE;
  const methods = Form.use<typeof createFormSchema>({
    schema: createFormSchema,
    values: {
      name: "",
      light: xcolor.construct(linked?.light ?? value),
      dark: !dark
        ? undefined
        : linked == null
          ? value
          : linked.dark == null
            ? undefined
            : xcolor.construct(linked.dark),
    },
  });
  const { updateAsync: create, variant } = useCreate();
  const creating = variant === "loading";
  const handleCreate = (): void => {
    if (creating || !methods.validate()) return;
    const { name, light, dark: darkValue } = methods.value();
    const key = uuid.create();
    void create({
      key,
      name: name.trim(),
      light,
      dark: darkValue,
      ...(palette === NO_PALETTE ? {} : { group: palette }),
    }).then((created) => {
      if (created) back(() => link(key));
    });
  };
  return (
    <Form.Form<typeof createFormSchema> {...methods}>
      <Fields
        onEnter={handleCreate}
        palette={
          <Select.Simple<group.Key>
            resourceName="palette"
            searchHidden
            value={palette}
            onChange={setPicked}
            triggerProps={{ size: "small" }}
            grow
            footer={
              <Select.Item<group.Key> itemKey={NO_PALETTE} textValue="No palette">
                <Icon.None />
                No palette
              </Select.Item>
            }
          >
            {palettes.map(({ key, name }) => (
              <Select.Item key={key} itemKey={key}>
                {name}
              </Select.Item>
            ))}
          </Select.Simple>
        }
        footer={
          <Button.Button
            variant="filled"
            size="small"
            status={creating ? "loading" : undefined}
            onClick={handleCreate}
          >
            Create
          </Button.Button>
        }
      />
    </Form.Form>
  );
};

/** The "+" that ends a palette. It opens a view that creates a color in the palette. */
const Add = (): ReactElement => {
  const { palette, focused } = Color.usePalette();
  const { open } = Dialog.useViews();
  const plus = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (focused) plus.current?.focus();
  }, [focused]);
  return (
    <Button.Button
      ref={plus}
      variant="text"
      square
      textColor={9}
      tooltip="Create color"
      aria-label={`Create a color in ${palette.name}`}
      onClick={(e) =>
        open("New color", <CreateView palette={palette.key} />, e.currentTarget)
      }
    >
      <Icon.Add />
    </Button.Button>
  );
};

interface PaletteAddProps {
  palette: group.Key;
}

const PaletteAdd = ({ palette }: PaletteAddProps): ReactElement | null =>
  Access.useUpdateGranted(group.resourceID(palette)) ? <Add /> : null;

/**
 * Ends a palette with a "+" that opens a view to create a named color at its end. The
 * Series palette and palettes that the user cannot update get none.
 */
export const CreateSwatch = (): ReactElement | null => {
  const { palette } = Color.usePalette();
  if (palette.key === color.SERIES_PALETTE) return null;
  return palette.key === NO_PALETTE ? <Add /> : <PaletteAdd palette={palette.key} />;
};

/**
 * Ends the Custom tab of a color picker. It opens a view that creates a named color
 * from the custom color.
 */
export const SaveToLibrary = (): ReactElement => {
  const { open } = Dialog.useViews();
  return (
    <Button.CreateListItem
      size="small"
      onClick={(e) => open("New color", <CreateView />, e.currentTarget)}
    >
      Save to library
    </Button.CreateListItem>
  );
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
  return naming ? (
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
      autoFocus={focused}
      onClick={() => {
        setFocused(false);
        setNaming(true);
      }}
    >
      Create palette
    </Button.CreateListItem>
  );
};
