// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/color/Settings.css";

import { color, group } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { theme } from "@synnaxlabs/lyra/theme";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Color as PColor, Flux, Group } from "@synnaxlabs/pluto";
import { array, color as xcolor, compare, type record, uuid } from "@synnaxlabs/x";
import { generateKeyBetween, generateNKeysBetween } from "fractional-indexing";
import pluralize from "pluralize";
import {
  createContext,
  type DragEvent,
  type KeyboardEvent,
  type ReactElement,
  use,
  useState,
} from "react";

import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";

const COLUMNS: List.Column[] = [
  { name: "Name", width: "minmax(24rem, 2fr)" },
  { name: "Light", width: "minmax(16rem, 1fr)" },
  { name: "Dark", width: "minmax(16rem, 1fr)" },
];

const PALETTE_PREFIX = "palette:";

interface PaletteNameProps {
  /** The confirm button's tooltip. */
  saveLabel: string;
  /** Saves the name, and resolves to whether it saved. */
  onSave: (name: string) => Promise<boolean>;
  onClose: () => void;
}

/** The cells of a field that names a palette. Enter saves, and Escape closes. */
const PaletteName = ({
  saveLabel,
  onSave,
  onClose,
}: PaletteNameProps): ReactElement => {
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const handleSave = (): void => {
    const trimmed = name.trim();
    if (trimmed === "" || saving) return;
    setSaving(true);
    void (async () => {
      const saved = await onSave(trimmed);
      setSaving(false);
      if (saved) onClose();
    })();
  };
  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSave();
    } else if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  };
  return (
    <>
      <List.Cell>
        <Input.Text
          value={name}
          onChange={setName}
          placeholder="Palette name"
          aria-label="Palette name"
          autoFocus
          onFocus={(e) => e.currentTarget.select()}
          full="x"
          onKeyDown={handleKeyDown}
        />
      </List.Cell>
      <List.Cell />
      <List.Cell>
        <Flex.Box x gap="small" justify="end">
          <Button.Button
            variant="filled"
            size="small"
            tooltip={saveLabel}
            aria-label={saveLabel}
            status={saving ? "loading" : undefined}
            onClick={handleSave}
          >
            <Icon.Check />
          </Button.Button>
          <Button.Button
            variant="text"
            size="small"
            tooltip="Cancel"
            aria-label="Cancel"
            onClick={onClose}
          >
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      </List.Cell>
    </>
  );
};

const paletteNameID = (palette: group.Key): string =>
  List.itemNameID(`${PALETTE_PREFIX}${palette}`);

interface PaletteTitleProps {
  palette: group.Key;
  name: string;
}

const PaletteTitle = ({ palette, name }: PaletteTitleProps): ReactElement => {
  const canUpdate = Access.useUpdateGranted(group.resourceID(palette));
  const { update } = Group.useRename();
  return (
    // A span takes the size and weight of the heading's label.
    <Text.MaybeEditable
      id={paletteNameID(palette)}
      defaultEl="span"
      value={name}
      onChange={canUpdate && ((next) => update({ key: palette, name: next }))}
      allowDoubleClick={false}
      overflow="fade"
    />
  );
};

interface PaletteHeadingProps {
  /** The key of the palette, or an empty string for colors in none. */
  palette: string;
  name: string;
  count: number;
  builtin: boolean;
  first: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Opens a form for a new color at the start of the palette. */
  onCreate?: () => void;
}

const PaletteHeading = ({
  palette,
  name,
  count,
  builtin,
  first,
  open,
  onOpenChange,
  onCreate,
}: PaletteHeadingProps): ReactElement => (
  <div
    className={CSS.cls(
      CSS.BE("color-settings", "palette"),
      first && CSS.BEM("color-settings", "palette", "first"),
      !open && CSS.BEM("color-settings", "palette", "closed"),
      palette !== "" && Menu.CONTEXT_TARGET,
    )}
    data-menu-key={palette === "" ? undefined : `${PALETTE_PREFIX}${palette}`}
  >
    <Header.Group
      size="large"
      open={open}
      onOpenChange={onOpenChange}
      className={CSS.BE("color-settings", "bar")}
      actions={
        onCreate == null ? undefined : (
          <Button.Button
            variant="text"
            size="small"
            textColor={9}
            tooltip="Create color"
            aria-label={`Create a color in ${name}`}
            onClick={onCreate}
          >
            <Icon.Add />
          </Button.Button>
        )
      }
    >
      {palette === "" ? name : <PaletteTitle palette={palette} name={name} />}
      <Text.Text level="tiny" color={9}>
        {count}
      </Text.Text>
      {builtin && <Badge.Badge>Built-in</Badge.Badge>}
    </Header.Group>
  </div>
);

interface ValueProps {
  value?: xcolor.Crude;
  fallback?: xcolor.Crude;
  onChange?: (value?: xcolor.Color) => void;
  label: string;
}

const Value = ({ value, fallback, onChange, label }: ValueProps): ReactElement => (
  <Flex.Box x align="center" gap="small">
    {fallback == null ? (
      <Color.Swatch
        value={value ?? "#000000"}
        onChange={onChange}
        size="small"
        aria-label={label}
      />
    ) : (
      <Color.Swatch
        value={value}
        fallback={fallback}
        onChange={onChange}
        size="small"
        aria-label={label}
      />
    )}
    <Text.Text level="small" variant="code" color={value == null ? 8 : 10}>
      {value == null ? "Auto" : xcolor.hex(value).toUpperCase()}
    </Text.Text>
  </Flex.Box>
);

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);

const path = PColor.formPath;

/** @returns whether a stored built-in differs from its shipped name or values. */
const edited = (stored: color.Named, shipped: color.Named): boolean =>
  stored.name !== shipped.name ||
  !xcolor.equals(stored.light, shipped.light) ||
  !xcolor.equals(stored.dark, shipped.dark);

/** Orders by code point, as the Core does. A color with no position comes last. */
const comparePositions = (a?: string, b?: string): number => {
  if (a === b) return 0;
  if (a == null) return 1;
  if (b == null) return -1;
  return a < b ? -1 : 1;
};

interface ContextValue {
  shipped: Map<color.Key, color.Named>;
  palettes: Map<string, string>;
  /** The colors in each palette, in order. */
  members: Map<string, color.Named[]>;
  /** Moves the dragged color before the target, in the target's palette. */
  move: (dragged: color.Key, target: color.Key) => void;
}

const Context = createContext<ContextValue>({
  shipped: new Map(),
  palettes: new Map(),
  members: new Map(),
  move: () => {},
});
Context.displayName = "Color.Settings";

const HAUL_TYPE = "palette_color";
const canDropColor = Haul.canDropOfType(HAUL_TYPE);
const FACTS = ["Color"];

/**
 * Returns the moves that put the dragged color just before the target, given every
 * color of their palette in order. When no position fits before the target, as when two
 * colors share one or the target has none, every color takes a new position.
 */
const placeBefore = (
  palette: color.Named[],
  dragged: color.Key,
  target: color.Key,
): PColor.MoveParams[] => {
  const others = array.removeKeyed(palette, dragged);
  const i = others.findIndex((c) => c.key === target);
  const next = others[i].position;
  const prev = i > 0 ? others[i - 1].position : undefined;
  if (next != null && (prev == null || prev < next))
    return [{ key: dragged, position: generateKeyBetween(prev ?? null, next) }];
  const order = others.map((c) => c.key);
  order.splice(i, 0, dragged);
  const positions = generateNKeysBetween(null, null, order.length);
  const current = new Map(palette.map((c) => [c.key, c.position]));
  return order.flatMap((key, j) =>
    current.get(key) === positions[j] ? [] : [{ key, position: positions[j] }],
  );
};

const Item = (props: List.ItemProps<color.Key>): ReactElement | null => {
  const { itemKey } = props;
  const named = List.useItem<color.Key, color.Named>(itemKey);
  const { shipped, move } = use(Context);
  const { form } = PColor.useForm({ query: { key: itemKey } });
  const name = Form.useField<string>(path.name, { ctx: form });
  const light = Form.useField<xcolor.Crude, xcolor.Color>(path.light, { ctx: form });
  const dark = Form.useField<xcolor.Crude | undefined, xcolor.Color | undefined>(
    path.dark,
    { ctx: form, optional: true },
  );
  const canUpdate = Access.useUpdateGranted(color.resourceID(itemKey));
  const itemClass = PlatformSettings.useItemClass(itemKey);
  const [over, setOver] = useState(false);
  const { startDrag, onDragEnd, onDragOver, onDrop } = Haul.useDragAndDrop({
    type: HAUL_TYPE,
    key: itemKey,
    canDrop: canDropColor,
    onDragOver: () => setOver(true),
    onDrop: ({ items }) => {
      setOver(false);
      const dragged = items.find(({ type }) => type === HAUL_TYPE);
      if (dragged != null && dragged.key !== itemKey)
        move(String(dragged.key), itemKey);
      return items;
    },
  });
  if (named == null) return null;
  const builtin = shipped.get(itemKey);
  const lightValue = light.value ?? named.light;
  const handleDragStart = (event: DragEvent): void =>
    startDrag({
      event,
      items: [{ type: HAUL_TYPE, key: itemKey }],
      preview: (
        <Haul.Preview
          swatch={xcolor.cssString(lightValue)}
          name={name.value}
          facts={FACTS}
        />
      ),
    });
  return (
    <Select.Item<color.Key>
      {...props}
      checkable
      align="center"
      aria-label={named.name}
      draggable={canUpdate}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDragLeave={() => setOver(false)}
      onDrop={onDrop}
      className={CSS.cls(
        CSS.BE("color-settings", "item"),
        over && CSS.BEM("color-settings", "item", "over"),
        itemClass,
      )}
    >
      <List.Cells>
        <List.Cell>
          <Flex.Box x align="center" gap="medium">
            <Text.MaybeEditable
              id={List.itemNameID(itemKey)}
              value={name.value}
              onChange={canUpdate && name.onChange}
              allowDoubleClick={false}
              overflow="fade"
            />
            {builtin != null && edited(named, builtin) && (
              <Tooltip.Dialog location="bottom">
                <Text.Text level="small">
                  This built-in color differs from the values that Synnax ships.
                </Text.Text>
                <Badge.Badge status="warning">Edited</Badge.Badge>
              </Tooltip.Dialog>
            )}
          </Flex.Box>
        </List.Cell>
        <List.Cell>
          <Value
            value={lightValue}
            onChange={canUpdate ? (v) => v != null && light.onChange(v) : undefined}
            label="Light theme color"
          />
        </List.Cell>
        <List.Cell>
          <Value
            value={dark?.value}
            fallback={theme.legible(lightValue, DARK)}
            onChange={canUpdate ? dark?.onChange : undefined}
            label="Dark theme color"
          />
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const createPath = PColor.createFormPath;

const NEW_PREFIX = "new:";

interface NewColorProps {
  /** The palette of the new color, or an empty string for none. */
  group: string;
  /** Every color in the palette, in order. */
  colors: color.Named[];
  /** The light value the next new color starts with. */
  light: xcolor.Color;
  onClose: () => void;
}

interface NewColorFormProps extends Omit<NewColorProps, "colors"> {
  /** Returns the position of the color that the form creates when it saves. */
  position: () => string;
  /** Called with the key of each color that the form creates. */
  onCreated: (key: record.Key) => void;
}

interface NewColorNameProps extends Input.Control<string> {
  onSave: () => void;
}

const NewColorName = ({ onSave, ...rest }: NewColorNameProps): ReactElement => {
  const onKeyDown = PlatformSettings.useCreateKeyDown(onSave);
  return (
    <Input.Text
      placeholder="Color name"
      aria-label="Color name"
      autoFocus
      full="x"
      onKeyDown={onKeyDown}
      {...rest}
    />
  );
};

const NewColorForm = ({
  group,
  position,
  light,
  onCreated,
  onClose,
}: NewColorFormProps): ReactElement => {
  const { form, save, variant } = PColor.useCreateForm({
    query: null,
    initialValues: { name: "", light, ...(group === "" ? {} : { group }) },
    beforeValidate: ({ set }) => set(createPath.position, position()),
    afterSave: ({ value }) => {
      const { key } = value();
      if (key != null) onCreated(key);
    },
  });
  const handleSave = (): void => {
    if (variant !== "loading") save();
  };
  return (
    <PlatformSettings.CreateProvider close={onClose} created={onCreated}>
      <Form.Form<typeof PColor.createFormSchema> {...form}>
        <div className={CSS.BE("color-settings", "form")}>
          <span className={CSS.BE("color-settings", "form-check")} />
          <List.Cells>
            <List.Cell>
              <Form.Field
                path={createPath.name}
                showLabel={false}
                padHelpText={false}
                grow
              >
                {(p) => <NewColorName onSave={handleSave} {...p} />}
              </Form.Field>
            </List.Cell>
            <List.Cell>
              <Form.Field path={createPath.light} showLabel={false} padHelpText={false}>
                {({ value, onChange }) => (
                  <Value
                    value={value}
                    onChange={(v) => v != null && onChange(v)}
                    label="Light theme color"
                  />
                )}
              </Form.Field>
            </List.Cell>
            <List.Cell>
              <Flex.Box x align="center" justify="between">
                <Text.Text level="small" variant="code" color={8}>
                  Auto
                </Text.Text>
                <PlatformSettings.CreateActions onSave={handleSave} status={variant} />
              </Flex.Box>
            </List.Cell>
          </List.Cells>
        </div>
      </Form.Form>
    </PlatformSettings.CreateProvider>
  );
};

/**
 * A form at the start of a palette. Each color it creates goes after the one it created
 * before, so the colors keep the order the user typed them in.
 */
const NewColor = ({ colors, ...rest }: NewColorProps): ReactElement => {
  const [last, setLast] = useState<record.Key | null>(null);
  const position = (): string => {
    const after = colors.find((c) => c.key === last)?.position ?? null;
    const next = colors.find(
      (c) => c.position != null && (after == null || c.position > after),
    );
    return generateKeyBetween(after, next?.position ?? null);
  };
  return <NewColorForm key={last} position={position} onCreated={setLast} {...rest} />;
};

interface NewPaletteProps {
  /** Called with the key of the new palette once the Core holds it. */
  onCreated: (key: group.Key) => void;
}

const NewPalette = ({ onCreated }: NewPaletteProps): ReactElement => {
  const { close } = PlatformSettings.useCreate();
  const { updateAsync } = Group.useCreate();
  const handleSave = async (name: string): Promise<boolean> => {
    const key = uuid.create();
    const created = await updateAsync({
      key,
      name,
      scope: color.TYPE_RESOURCE_ID.type,
    });
    if (created) onCreated(key);
    return created;
  };
  return (
    <PlatformSettings.CreateFrame>
      <PaletteName saveLabel="Create" onSave={handleSave} onClose={close} />
    </PlatformSettings.CreateFrame>
  );
};

interface PaletteMenuProps {
  palette: group.Key;
}

const PaletteMenu = ({ palette }: PaletteMenuProps): ReactElement => {
  const { palettes, members, shipped } = use(Context);
  const colors = members.get(palette) ?? [];
  const keys = colors.map((c) => c.key);
  const id = group.resourceID(palette);
  const canUpdate = Access.useUpdateGranted(id);
  const canDeletePalette = Access.useDeleteGranted(id);
  const canDeleteColors = Access.useDeleteGranted(color.resourceID(keys));
  const also = `This also deletes its ${pluralize("color", colors.length, true)}.`;
  const confirm = Modals.useConfirmDelete({
    type: "Palette",
    icon: "Swatches",
    description:
      colors.length === 0 ? undefined : `${also} This action cannot be undone.`,
  });
  const { update: remove } = PColor.useDeletePalette({
    beforeUpdate: async () => await confirm({ name: palettes.get(palette) ?? "" }),
  });
  return (
    <ContextMenu.Menu>
      {canUpdate && (
        <ContextMenu.RenameItem onClick={() => Text.edit(paletteNameID(palette))} />
      )}
      <Menu.Divider />
      {canDeletePalette &&
        canDeleteColors &&
        !colors.some((c) => shipped.has(c.key)) && (
          <ContextMenu.DeleteItem onClick={() => remove({ palette, colors: keys })} />
        )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const ColorsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<color.Key, color.Named>();
  const { shipped } = use(Context);
  const colors = getItem?.(keys) ?? [];
  const ids = color.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const deletable = colors.filter((c) => !shipped.has(c.key));
  const confirm = Modals.useConfirmDelete({ type: "Color", icon: "Swatches" });
  const { update: remove } = PColor.useDelete({
    beforeUpdate: async () => await confirm(deletable),
  });
  const { update: restore } = PColor.useRestore();
  const single = colors.length === 1 ? colors[0] : undefined;
  const restorable = colors.flatMap((c) => {
    const s = shipped.get(c.key);
    return s != null && edited(c, s) ? [s] : [];
  });
  return (
    <ContextMenu.Menu>
      {single != null && canUpdate && (
        <ContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(single.key))}
        />
      )}
      {restorable.length > 0 && canUpdate && (
        <Menu.Item
          itemKey="restore"
          onClick={() => restorable.forEach((s) => restore(s))}
        >
          <Icon.Reset />
          Restore default
        </Menu.Item>
      )}
      <Menu.Divider />
      {deletable.length > 0 && canDelete && (
        <ContextMenu.DeleteItem onClick={() => remove(deletable.map((c) => c.key))} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const SettingsMenu = (props: Menu.ContextMenuMenuProps): ReactElement => {
  const [first] = props.keys;
  if (props.keys.length === 1 && first.startsWith(PALETTE_PREFIX))
    return <PaletteMenu palette={first.slice(PALETTE_PREFIX.length)} />;
  return <ColorsMenu {...props} />;
};

const menu = Component.renderProp(SettingsMenu);

/**
 * Every named color, by palette and then by place in the palette. Each line renames
 * and recolors its color in place, a drag moves a color within its palette, and the
 * menu restores built-ins or deletes other colors.
 */
export const Settings = (): ReactElement => {
  const list = Flux.useList((client) => client.colors, {
    name: "colors",
    query: {},
    pageSize: 1000,
  });
  const records = PColor.useResultList({});
  const builtins = PColor.useResultBuiltins({});
  const groups = Group.useResultList({ where: { scope: { eq: "color" } } });
  const { update: moveColor } = PColor.useMove();
  const canCreate = Access.useCreateGranted(color.TYPE_RESOURCE_ID);
  const canCreatePalette = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  const [closed, setClosed] = useState<ReadonlySet<string>>(() => new Set());
  const [creating, setCreating] = useState<string | null>(null);
  const all = records.variant === "success" ? records.data : [];
  const byKey = new Map(all.map((c) => [c.key, c]));
  const palettes = new Map(
    (groups.variant === "success" ? groups.data : []).map((g) => [g.key, g.name]),
  );
  const shipped = new Map(
    (builtins.variant === "success" ? builtins.data : []).map((c) => [c.key, c]),
  );
  const paletteName = (c: color.Named): string | undefined =>
    c.group == null ? undefined : palettes.get(c.group);
  const byPalette = (a: color.Named, b: color.Named): number => {
    const pa = paletteName(a);
    const pb = paletteName(b);
    if (pa === pb) return 0;
    if (pa == null) return 1;
    if (pb == null) return -1;
    return compare.text(pa, pb);
  };
  const inOrder = compare.chain(
    byPalette,
    (a, b) => comparePositions(a.position, b.position),
    compare.by((c) => c.name),
  );
  const sorted = list.data
    .flatMap((k) => {
      const c = byKey.get(k);
      return c == null ? [] : [c];
    })
    .sort(inOrder);
  // Every color of each palette, whatever the search shows.
  const members = Map.groupBy([...all].sort(inOrder), (c) => c.group ?? "");
  const move = (dragged: color.Key, target: color.Key): void => {
    const d = byKey.get(dragged);
    const t = byKey.get(target);
    if (d == null || t == null || d.group !== t.group) return;
    moveColor(placeBefore(members.get(t.group ?? "") ?? [], dragged, target));
  };
  const shippedGroups = new Set([...shipped.values()].map((c) => c.group));
  const series = [...shipped.values()]
    .sort((a, b) => comparePositions(a.position, b.position))
    .map((c) => c.light);
  const byGroup = Map.groupBy(sorted, (c) => c.group ?? "");
  const order = [...palettes.entries()]
    .sort(compare.by(([, name]) => name))
    .map(([key]) => key);
  if (byGroup.has("")) order.push("");
  const headings = new Map<string, PaletteHeadingProps>();
  const news = new Map<string, NewColorProps>();
  const data: string[] = [];
  const setOpen = (palette: string, open: boolean): void =>
    setClosed((prev) => {
      const next = new Set(prev);
      if (open) next.delete(palette);
      else next.add(palette);
      return next;
    });
  for (const palette of order) {
    const colors = byGroup.get(palette) ?? [];
    const heading = `${PALETTE_PREFIX}${palette}`;
    const open = !closed.has(palette);
    const name = palettes.get(palette) ?? "No palette";
    headings.set(heading, {
      palette,
      name,
      count: colors.length,
      builtin: shippedGroups.has(palette),
      first: data.length === 0,
      open,
      onOpenChange: (next) => setOpen(palette, next),
      onCreate:
        canCreate && series.length > 0
          ? () => {
              setOpen(palette, true);
              setCreating(palette);
            }
          : undefined,
    });
    data.push(heading);
    if (!open) continue;
    if (creating === palette) {
      const key = `${NEW_PREFIX}${palette}`;
      const inPalette = members.get(palette) ?? [];
      const unused = series.find(
        (v) => !inPalette.some((c) => xcolor.equals(c.light, v)),
      );
      news.set(key, {
        group: palette,
        colors: inPalette,
        light: unused ?? series[inPalette.length % series.length],
        onClose: () => setCreating(null),
      });
      data.push(key);
    }
    data.push(...colors.map((c) => c.key));
  }
  return (
    <Context value={{ shipped, palettes, members, move }}>
      <PlatformSettings.Table<color.Key, color.Named>
        list={{ ...list, data }}
        columns={COLUMNS}
        noun="colors"
        menu={menu}
        createLabel={canCreate && canCreatePalette ? "Create palette" : undefined}
        create={<NewPalette onCreated={setCreating} />}
      >
        {({ key, ...p }) => {
          const heading = headings.get(p.itemKey);
          if (heading != null) return <PaletteHeading key={key} {...heading} />;
          const n = news.get(p.itemKey);
          if (n != null) return <NewColor key={key} {...n} />;
          return <Item key={key} {...p} />;
        }}
      </PlatformSettings.Table>
    </Context>
  );
};
