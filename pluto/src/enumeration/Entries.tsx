// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/enumeration/Entries.css";

import { color, enumeration } from "@synnaxlabs/client";
import { Color as BaseColor } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { color as xcolor, deep, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Access } from "@/access";
import { Color } from "@/color";
import { CSS } from "@/css";
import { check, type ResolveColor } from "@/enumeration/check";
import { formPath, useForm } from "@/enumeration/queries";

const NAME = 0;
const VALUE = 1;

/** Where the color and abnormal cells sit in a row, or -1 for a hidden column. */
interface Layout {
  color: number;
  abnormal: number;
}

const layoutOf = (colorHidden: boolean, abnormalHidden: boolean): Layout => ({
  color: colorHidden ? -1 : 2,
  abnormal: abnormalHidden ? -1 : colorHidden ? 2 : 3,
});

/** Reads a fixed color as the hex box does, Auto from empty text, or null otherwise. */
const parseColor = (text: string): color.Ref | undefined | null => {
  if (text.trim() === "") return undefined;
  const parsed = BaseColor.parseHexInput(text, 1);
  return parsed == null ? null : { type: "fixed", value: parsed };
};

const toRow = (e: enumeration.Entry, layout: Layout): Input.TableCell[] => {
  const row: Input.TableCell[] = [e.name, e.value];
  if (layout.color >= 0) row[layout.color] = e.color;
  if (layout.abnormal >= 0) row[layout.abnormal] = e.abnormal;
  return row;
};

/** @returns the entry that the row shows. A hidden field keeps its value from prev. */
const fromRow = (
  row: Input.TableCell[],
  layout: Layout,
  prev: Pick<enumeration.Entry, "key"> & Partial<enumeration.Entry>,
): enumeration.Entry => {
  const ref =
    layout.color < 0 ? prev.color : color.refZ.optional().parse(row[layout.color]);
  return {
    key: prev.key,
    name: String(row[NAME]),
    value: Number(row[VALUE]),
    abnormal:
      layout.abnormal < 0 ? (prev.abnormal ?? false) : row[layout.abnormal] === true,
    ...(ref != null && { color: ref }),
  };
};

/**
 * @returns the indices of the entries that must hold their shown value unsent. An entry
 * whose shown value differs from its stored one holds when another entry shows that
 * value, or stores it and holds too. A swap that clashes with no other entry holds
 * neither entry, so both save.
 */
const holding = (shown: number[], stored: (number | undefined)[]): Set<number> => {
  const held = new Set<number>();
  for (let grew = true; grew;) {
    grew = false;
    shown.forEach((v, i) => {
      if (held.has(i) || v === stored[i]) return;
      const clash = shown.some(
        (w, j) => j !== i && (w === v || (held.has(j) && stored[j] === v)),
      );
      if (!clash) return;
      held.add(i);
      grew = true;
    });
  }
  return held;
};

/** @returns the least whole number above every value of the rows, or 0. */
const nextValue = (rows: Input.TableCell[][]): number =>
  rows.length === 0
    ? 0
    : Math.max(...rows.map((r) => Math.floor(Number(r[VALUE])))) + 1;

interface WarningsProps {
  warnings: string[];
}

const Warnings = ({ warnings }: WarningsProps): ReactElement => (
  <Tooltip.Dialog location="bottom">
    <Flex.Box y gap="tiny">
      {warnings.map((w) => (
        <Text.Text key={w} level="small">
          {w}
        </Text.Text>
      ))}
    </Flex.Box>
    <span
      role="img"
      aria-label={warnings.join(" ")}
      className={CSS.BE("entries", "warning")}
    >
      <Status.Indicator variant="warning">
        <Icon.Warning />
      </Status.Indicator>
    </span>
  </Tooltip.Dialog>
);

interface ColorCellProps extends Input.TableCellProps<color.Ref | undefined> {
  /** The name of a named color, or the hex digits of a fixed one. */
  text: string;
  warnings: string[];
}

const ColorCell = ({
  value,
  onChange,
  preview,
  variant,
  "aria-label": label,
  text,
  warnings,
}: ColorCellProps): ReactElement => (
  <BaseColor.HexText
    text={text}
    named={value?.type === "named"}
    alpha={value?.type === "fixed" ? xcolor.opacity(value.value) : 1}
    placeholder="Auto"
    variant={variant}
    aria-label={label}
    preview={preview}
    onlyChangeOnBlur
    startContent={
      <>
        <Color.Swatch
          value={value}
          onChange={onChange}
          fallback={xcolor.TRANSPARENT}
          disabled={preview}
          onlyChangeOnBlur
          size="small"
          aria-label={`Pick ${label.toLowerCase()}`}
        />
        {value?.type === "fixed" && "#"}
      </>
    }
    endContent={warnings.length > 0 ? <Warnings warnings={warnings} /> : undefined}
    onChange={(next) => onChange({ type: "fixed", value: next })}
    onClear={() => onChange(undefined)}
  />
);

/** One edit of the entries, which the caller saves as one change. */
export interface Edit {
  /** The entries whose fields change, such as the two entries of a swap of values. */
  changed: enumeration.Entry[];
  /** The entries to add, in order, after the last entry. */
  added: enumeration.Entry[];
  /** The keys of the entries to remove. */
  removed: string[];
}

export interface TableProps extends Pick<Input.TableProps, "autoFocus"> {
  value: enumeration.Entry[];
  /** Called once per edit, since a swap of two values is valid only as a whole. */
  onChange: (edit: Edit) => void;
  onMove: (from: number, to: number) => void;
  /** Shows the entries read-only. */
  preview?: boolean;
  /** Leaves out the Abnormal column, for entries that cannot be abnormal. */
  abnormalHidden?: boolean;
  /** Leaves out the Color column, for entries whose color draws nowhere. */
  colorHidden?: boolean;
}

/**
 * Edits a list of entries as a grid of name, value, color, and abnormal. A value that
 * another entry has stays in its cell, marked and unsent. It saves when the user's own
 * edit frees it, and the cell shows the stored value again when another user frees it.
 * An entry that is hard to tell apart from the others shows a warning in its color.
 */
export const Table = ({
  value,
  onChange,
  onMove,
  preview,
  abnormalHidden = false,
  colorHidden = false,
  autoFocus,
}: TableProps): ReactElement => {
  const layout = layoutOf(colorHidden, abnormalHidden);
  const [held, setHeld] = useState<ReadonlyMap<string, number>>(() => new Map());
  const resolve = Color.useResolve();
  const colors = Color.useResultList({});
  const named = new Map(
    (colors.variant === "success" ? colors.data : []).map((c) => [c.key, c]),
  );
  const resolvePair: ResolveColor = ({ color: ref }) => {
    if (ref == null) return undefined;
    if (ref.type === "fixed") return { light: ref.value };
    const n = named.get(ref.key);
    return n == null ? undefined : { light: n.light, dark: n.dark };
  };
  const colorText = (ref: color.Ref | undefined): string => {
    if (ref == null) return "";
    if (ref.type === "named") return named.get(ref.key)?.name ?? "";
    return xcolor.hex(ref.value).slice(1);
  };
  const withHeld = (holds: ReadonlyMap<string, number>): enumeration.Entry[] =>
    value.map((e) => {
      const v = holds.get(e.key);
      return v == null ? e : { ...e, value: v };
    });
  const holderIn = (
    entries: enumeration.Entry[],
    key: string,
    v: number,
  ): enumeration.Entry | undefined =>
    entries.find((e) => e.key !== key && e.value === v);
  const stored = value.map((e) => e.value);
  // A hold lasts only while it clashes. Another user can free it.
  const asHeld = withHeld(held);
  const lasting = holding(
    asHeld.map((e) => e.value),
    stored,
  );
  const holds = new Map(
    asHeld.filter((_, i) => lasting.has(i)).map((e) => [e.key, e.value]),
  );
  // A freed hold goes for good, so a later clash does not bring it back.
  if (holds.size !== held.size) setHeld(holds);
  const shown = withHeld(holds);
  // Counts stored values too, since a held entry shows a value it does not store.
  const freeValue = (rows: Input.TableCell[][]): number =>
    nextValue([...rows, ...value.map((e) => toRow(e, layout))]);

  const handleChange = (rows: Input.TableCell[][]): void => {
    const changed: enumeration.Entry[] = [];
    const added: enumeration.Entry[] = [];
    const nextHeld = new Map<string, number>();
    // An edit can free a value that another row holds, so each row checks the rows as
    // they show after the edit.
    const clashing = holding(
      rows.map((row) => Number(row[VALUE])),
      rows.map((_, i) => value.at(i)?.value),
    );
    rows.forEach((row, i) => {
      const prev = value.at(i);
      let next = fromRow(row, layout, prev ?? { key: id.create() });
      if (prev == null) {
        // A pasted entry saves with a free value and holds the one it repeats.
        if (clashing.has(i)) {
          nextHeld.set(next.key, next.value);
          next = {
            ...next,
            value: freeValue([...rows, ...added.map((e) => toRow(e, layout))]),
          };
        }
        added.push(next);
        return;
      }
      if (clashing.has(i)) {
        nextHeld.set(prev.key, next.value);
        next = { ...next, value: prev.value };
      }
      if (!deep.equal(next, prev)) changed.push(next);
    });
    if (!deep.equal([...nextHeld], [...held])) setHeld(nextHeld);
    if (changed.length > 0 || added.length > 0)
      onChange({ changed, added, removed: [] });
  };

  const handleRemove = (i: number): void => {
    const rest = shown.filter((_, j) => j !== i);
    const still = holding(
      rest.map((e) => e.value),
      stored.filter((_, j) => j !== i),
    );
    const nextHeld = new Map<string, number>();
    const freed: enumeration.Entry[] = [];
    rest.forEach((entry, j) => {
      if (!holds.has(entry.key)) return;
      if (still.has(j)) nextHeld.set(entry.key, entry.value);
      else freed.push(entry);
    });
    setHeld(nextHeld);
    onChange({ changed: freed, added: [], removed: [shown[i].key] });
  };

  const cellStatus = ({ x, y }: xy.XY): Status.Crude | undefined => {
    const entry = shown[y];
    if (x !== VALUE || entry == null) return undefined;
    const v = entry.value;
    // Node options from before enumerations can store a value twice.
    const repeated = holds.has(entry.key) || holderIn(value, entry.key, v) != null;
    const other = shown.find(
      (e, j) => j !== y && (e.value === v || (holds.has(e.key) && stored[j] === v)),
    );
    if (!repeated || other == null) return undefined;
    const name = other.name.trim() === "" ? "Another entry" : other.name;
    return {
      variant: "error",
      message: `${name} has the value ${v}. Each entry needs its own value.`,
    };
  };

  const warningsAt = (row: number): string[] => {
    const entry = value[row];
    if (entry == null) return [];
    return check(entry, value, resolvePair, { abnormalHidden });
  };

  const renderHaulPreview = (row: number): ReactElement => {
    const entry = value[row];
    const swatch = entry?.color == null ? undefined : resolve(entry.color);
    return (
      <Haul.Preview
        icon={<Icon.Enumeration />}
        swatch={swatch == null ? undefined : xcolor.cssString(swatch)}
        name={entry?.name ?? ""}
        facts={[`Value ${entry?.value}`]}
      />
    );
  };

  const colorColumn = (
    <Input.TableColumn<color.Ref | undefined>
      key="color"
      name="Color"
      type="custom"
      empty={undefined}
      parse={parseColor}
      width={24}
    >
      {(props) => (
        <ColorCell
          {...props}
          text={colorText(props.value)}
          warnings={warningsAt(props.row)}
        />
      )}
    </Input.TableColumn>
  );

  const abnormalColumn = (
    <Input.TableColumn key="abnormal" name="Abnormal" type="boolean" width={14}>
      {(props) => (
        // Each edit saves at once, so the cell is a switch.
        <Input.Switch
          value={props.value}
          onChange={props.onChange}
          variant={props.variant}
          disabled={props.preview}
          aria-label={props["aria-label"]}
        />
      )}
    </Input.TableColumn>
  );

  return (
    <div className={CSS.B("entries")}>
      <Input.Table
        value={shown.map((e) => toRow(e, layout))}
        onChange={handleChange}
        resourceName="entry"
        preview={preview}
        autoFocus={autoFocus}
        cellStatus={cellStatus}
        createRow={(rows) =>
          toRow({ key: "", name: "", value: freeValue(rows), abnormal: false }, layout)
        }
        onRemove={handleRemove}
        onMove={onMove}
        rowKey={(i) => shown[i].key}
        renderHaulPreview={renderHaulPreview}
      >
        {[
          <Input.TableColumn key="name" name="Name" type="string">
            {(props) => <Input.Text {...props} onlyChangeOnBlur />}
          </Input.TableColumn>,
          <Input.TableColumn key="value" name="Value" type="number" width={14} />,
          ...(colorHidden ? [] : [colorColumn]),
          ...(abnormalHidden ? [] : [abnormalColumn]),
        ]}
      </Input.Table>
    </div>
  );
};

export interface EntriesProps extends Pick<TableProps, "autoFocus"> {
  enumeration: enumeration.Key;
}

/**
 * Edits the entries of an enumeration, which every editor of it shares. Each edit saves
 * at once. A user who cannot update the enumeration sees it read-only.
 */
export const Entries = ({
  enumeration: key,
  autoFocus,
}: EntriesProps): ReactElement => {
  const { form } = useForm({ query: { key } });
  const entries = Form.useFieldValue<enumeration.Entry[]>(formPath.entries, {
    ctx: form,
  });
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(key));
  const handleChange = ({ changed, added, removed }: Edit): void =>
    form.batch(() => {
      for (const k of removed) form.remove(formPath.entries[k]);
      for (const { color, ...next } of changed) {
        const current = entries.find((e) => e.key === next.key);
        if (current == null) throw new Error(`no entry with key ${next.key}`);
        const { color: currentColor, ...base } = current;
        const at = formPath.entries[next.key];
        // Only the fields that differ go out, so another user's edit to the rest stays.
        Form.setChanged(form, at, base, next);
        if (deep.equal(color, currentColor)) continue;
        if (color == null) form.remove(at.color);
        else form.set(at.color, color);
      }
      let after = entries.findLast((e) => !removed.includes(e.key))?.key;
      for (const entry of added) {
        form.insert(formPath.entries, entry, after);
        after = entry.key;
      }
    });
  const handleMove = (from: number, to: number): void => {
    const moved = entries[from];
    const others = entries.filter((e) => e.key !== moved.key);
    form.move(formPath.entries, moved.key, to > 0 ? others[to - 1].key : undefined);
  };
  return (
    <Table
      value={entries}
      preview={!canUpdate}
      onChange={handleChange}
      onMove={handleMove}
      autoFocus={autoFocus}
    />
  );
};
