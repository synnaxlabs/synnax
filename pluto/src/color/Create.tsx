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
import { usePalettes } from "@/color/library";
import { createFormSchema, useAppend, useCreatePalette } from "@/color/queries";
import { Fields } from "@/color/View";
import { Synnax } from "@/synnax";

interface NameEntryProps {
  placeholder: string;
  /** Called with the trimmed name. Resolves to whether the create succeeded. */
  onSave: (name: string) => Promise<boolean>;
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
  // A ref, so that two key presses in one render cannot both save.
  const savingRef = useRef(false);
  const save = (): void => {
    if (savingRef.current) return;
    const trimmed = name.trim();
    if (trimmed === "") return onCancel();
    savingRef.current = true;
    void onSave(trimmed).then((ok) => {
      if (!ok) savingRef.current = false;
    });
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

interface NewColorProps {
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
const NewColor = ({ palette: initialPalette }: NewColorProps): ReactElement => {
  const { value, linked, onNamedChange } = Color.usePicker();
  const { back } = Dialog.useViews();
  const { dark } = Theming.use();
  const client = Synnax.use();
  const palettes = usePalettes().filter(
    ({ key }) =>
      key !== color.SERIES_PALETTE &&
      key !== "" &&
      Access.updateGranted({ client, id: group.resourceID(key) }),
  );
  const [palette, setPalette] = useState(initialPalette ?? palettes[0]?.key ?? "");
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
  const { updateAsync: append, variant } = useAppend();
  // A ref, so that two key presses in one render cannot both create.
  const creating = useRef(false);
  const create = (): void => {
    if (creating.current || !methods.validate()) return;
    creating.current = true;
    const { name, light, dark: darkValue } = methods.value();
    const key = uuid.create();
    void append({
      key,
      name: name.trim(),
      light: xcolor.construct(light),
      dark: darkValue == null ? undefined : xcolor.construct(darkValue),
      ...(palette === "" ? {} : { group: palette }),
    }).then((created) => {
      if (created) back(() => onNamedChange(key));
      else creating.current = false;
    });
  };
  return (
    <Form.Form<typeof createFormSchema> {...methods}>
      <Fields
        onEnter={create}
        palette={
          <Select.Simple<group.Key>
            resourceName="palette"
            searchHidden
            value={palette}
            onChange={(key: group.Key | undefined) => key != null && setPalette(key)}
            triggerProps={{ size: "small" }}
            grow
            footer={
              <Select.Item<group.Key> itemKey="" textValue="No palette">
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
            status={variant === "loading" ? "loading" : undefined}
            onClick={create}
          >
            Create
          </Button.Button>
        }
      />
    </Form.Form>
  );
};

interface AddProps {
  /** If true, the "+" takes the focus. */
  focused?: boolean;
  /** Called when the "+" takes the focus. */
  onFocus?: () => void;
}

/** The "+" that ends a palette. It opens a view that creates a color in the palette. */
const Add = ({ focused = false, onFocus }: AddProps): ReactElement => {
  const { palette } = Color.usePalette();
  const { open } = Dialog.useViews();
  const plus = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (focused) plus.current?.focus();
  }, [focused]);
  return (
    <Button.Button
      ref={plus}
      onFocus={onFocus}
      variant="text"
      square
      textColor={9}
      tooltip="Create color"
      aria-label={`Create a color in ${palette.name}`}
      onClick={(e) =>
        open("New color", <NewColor palette={palette.key} />, e.currentTarget)
      }
    >
      <Icon.Add />
    </Button.Button>
  );
};

interface PaletteAddProps extends AddProps {
  palette: group.Key;
}

const PaletteAdd = ({ palette, ...rest }: PaletteAddProps): ReactElement | null =>
  Access.useUpdateGranted(group.resourceID(palette)) ? <Add {...rest} /> : null;

export interface CreateSwatchProps extends Pick<AddProps, "onFocus"> {
  /** The palette whose "+" takes the focus, such as a new palette. */
  focusedPalette?: group.Key;
}

/**
 * Ends a palette of a color picker with a "+" that opens a view to create a named color
 * after the last positioned color of the palette. The Series palette gets none, because
 * its colors are the series of a chart. A palette that the user cannot update gets
 * none, because a new color updates it.
 */
export const CreateSwatch = ({
  focusedPalette,
  onFocus,
}: CreateSwatchProps): ReactElement | null => {
  const { palette } = Color.usePalette();
  if (palette.key === color.SERIES_PALETTE) return null;
  const focused = palette.key === focusedPalette;
  return palette.key === "" ? (
    <Add focused={focused} onFocus={onFocus} />
  ) : (
    <PaletteAdd palette={palette.key} focused={focused} onFocus={onFocus} />
  );
};

/**
 * Ends the Custom tab of a color picker. It opens a view that creates a named color
 * from the custom color.
 */
export const SaveColor = (): ReactElement => {
  const { open } = Dialog.useViews();
  return (
    <Button.CreateListItem
      size="small"
      onClick={(e) => open("New color", <NewColor />, e.currentTarget)}
    >
      Save to library
    </Button.CreateListItem>
  );
};

export interface CreatePaletteProps {
  /** Called with the key of a created palette that ends with a "+", for its focus. */
  onCreate?: (key: group.Key) => void;
}

/**
 * Ends the palettes of a color picker. It creates a palette after asking its name, and
 * clears the search so that the palette shows. A create moves the focus to the "+" of
 * the new palette. A cancel, or a create for a user who cannot add to the palette,
 * moves it back to the button that opened the entry.
 */
export const CreatePalette = ({ onCreate }: CreatePaletteProps): ReactElement => {
  const { clearSearch } = Color.usePicker();
  // No policy can name the key of a new palette, so only a grant on every palette gives
  // it a "+".
  const extendable = Access.useUpdateGranted(group.TYPE_RESOURCE_ID);
  const [naming, setNaming] = useState(false);
  const [focused, setFocused] = useState(false);
  const { updateAsync: create, variant } = useCreatePalette();
  const handleSave = async (name: string): Promise<boolean> => {
    const key = uuid.create();
    const created = await create({ key, name });
    if (!created) return false;
    setNaming(false);
    clearSearch();
    if (extendable) onCreate?.(key);
    else setFocused(true);
    return true;
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
