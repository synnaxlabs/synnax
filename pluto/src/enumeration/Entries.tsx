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
const COLOR = 2;
const ABNORMAL = 3;

const HEX = /^#?([0-9a-f]{6}(?:[0-9a-f]{2})?)$/i;

const toRow = (e: enumeration.Entry): Input.TableCell[] => [
  e.name,
  e.value,
  e.color,
  e.abnormal,
];

const fromRow = (row: Input.TableCell[], key: string): enumeration.Entry => {
  const ref = color.refZ.optional().parse(row[COLOR]);
  return {
    key,
    name: String(row[NAME]),
    value: Number(row[VALUE]),
    abnormal: row[ABNORMAL] === true,
    ...(ref != null && { color: ref }),
  };
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
  /** The name of a named color, or the hex code of a fixed one. */
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
  <Input.Text
    value={text}
    placeholder="Auto"
    variant={variant}
    aria-label={label}
    preview={preview}
    onlyChangeOnBlur
    startContent={
      <Color.Swatch
        value={value}
        onChange={onChange}
        fallback={xcolor.TRANSPARENT}
        disabled={preview}
        size="small"
        className={CSS.BE("entries", "swatch")}
        aria-label={`Pick ${label.toLowerCase()}`}
      />
    }
    endContent={warnings.length > 0 ? <Warnings warnings={warnings} /> : undefined}
    onChange={(next) => {
      if (next.trim() === "") return onChange(undefined);
      const hex = HEX.exec(next.trim())?.[1];
      if (hex != null) onChange({ type: "fixed", value: xcolor.construct(`#${hex}`) });
    }}
  />
);

interface EmptyCellProps {
  variant?: Input.Variant;
  "aria-label": string;
}

const EmptyCell = ({ variant, "aria-label": label }: EmptyCellProps): ReactElement => (
  <Input.Text
    value=""
    variant={variant}
    aria-label={label}
    disabled
    onChange={() => {}}
  />
);

export interface TableProps {
  value: enumeration.Entry[];
  /** Called with an entry the user edits. */
  onSet: (entry: enumeration.Entry) => void;
  /** Called with the entries the user adds, in order, after the last entry. */
  onAdd: (entries: enumeration.Entry[]) => void;
  onRemove: (index: number) => void;
  onMove: (from: number, to: number) => void;
  /** Shows the entries read-only. */
  preview?: boolean;
  /** Shows each Abnormal cell empty, for entries that cannot be abnormal. */
  abnormalDisabled?: boolean;
  /** Shows each Color cell empty, for entries whose color draws nowhere. */
  colorDisabled?: boolean;
  /** Focuses the first Name cell on mount, or Add entry when there are no entries. */
  autoFocus?: boolean;
}

/**
 * Edits a list of entries as a grid of name, value, color, and abnormal. A value that
 * another entry has stays in its cell, marked and unsent, until no other entry has it.
 * An entry that is hard to tell apart from the others shows a warning in its color.
 */
export const Table = ({
  value,
  onSet,
  onAdd,
  onRemove,
  onMove,
  preview,
  abnormalDisabled = false,
  colorDisabled = false,
  autoFocus,
}: TableProps): ReactElement => {
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
    return xcolor.hex(ref.value).slice(1).toUpperCase();
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
  // A hold lasts only while another entry has its value. Another user can free it.
  const holds = new Map(
    [...held].filter(([key, v]) => holderIn(withHeld(held), key, v) != null),
  );
  const shown = withHeld(holds);
  // Counts stored values too, since a held entry shows a value it does not store.
  const freeValue = (rows: Input.TableCell[][]): number =>
    nextValue([...rows, ...value.map(toRow)]);

  const handleChange = (rows: Input.TableCell[][]): void => {
    const added: enumeration.Entry[] = [];
    const nextHeld = new Map<string, number>();
    // An edit can free a value that another row holds, so each row checks the rows as
    // they show after the edit.
    const taken = (i: number, v: number): boolean =>
      rows.some((row, j) => j !== i && Number(row[VALUE]) === v);
    rows.forEach((row, i) => {
      const prev = value.at(i);
      let next = fromRow(row, prev?.key ?? id.create());
      if (prev == null) {
        // A pasted entry saves with a free value and holds the one it repeats.
        if (taken(i, next.value)) {
          nextHeld.set(next.key, next.value);
          next = { ...next, value: freeValue([...rows, ...added.map(toRow)]) };
        }
        added.push(next);
        return;
      }
      if (next.value !== prev.value && taken(i, next.value)) {
        nextHeld.set(prev.key, next.value);
        next = { ...next, value: prev.value };
      }
      if (!deep.equal(next, prev)) onSet(next);
    });
    if (!deep.equal([...nextHeld], [...held])) setHeld(nextHeld);
    if (added.length > 0) onAdd(added);
  };

  const handleRemove = (i: number): void => {
    onRemove(i);
    const rest = shown.filter((_, j) => j !== i);
    const nextHeld = new Map<string, number>();
    for (const [key, v] of holds) {
      const entry = rest.find((e) => e.key === key);
      if (entry == null) continue;
      if (holderIn(rest, key, v) == null) onSet(entry);
      else nextHeld.set(key, v);
    }
    setHeld(nextHeld);
  };

  const cellStatus = ({ x, y }: xy.XY): Status.Crude | undefined => {
    const entry = shown[y];
    if (x !== VALUE || entry == null) return undefined;
    const v = entry.value;
    // Node options from before enumerations can store a value twice.
    const repeated = holds.has(entry.key) || holderIn(value, entry.key, v) != null;
    const other = holderIn(shown, entry.key, v);
    if (!repeated || other == null) return undefined;
    const name = other.name.trim() === "" ? "Another entry" : other.name;
    return {
      variant: "error",
      message: `${name} has the value ${v}. Each entry needs its own value.`,
    };
  };

  const warningsAt = (row: number): string[] => {
    const entry = value[row];
    if (entry == null || colorDisabled) return [];
    return check(entry, value, resolvePair, { abnormalDisabled });
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

  return (
    <div className={CSS.B("entries")}>
      <Input.Table
        value={shown.map(toRow)}
        onChange={handleChange}
        resourceName="entry"
        preview={preview}
        autoFocus={autoFocus}
        cellStatus={cellStatus}
        createRow={(rows) => ["", freeValue(rows), undefined, false]}
        onRemove={handleRemove}
        onMove={onMove}
        renderHaulPreview={renderHaulPreview}
      >
        <Input.TableColumn name="Name" type="string" />
        <Input.TableColumn name="Value" type="number" width={14} />
        <Input.TableColumn<color.Ref | undefined>
          name="Color"
          type="custom"
          empty={undefined}
          width={24}
        >
          {(props) =>
            colorDisabled ? (
              <EmptyCell {...props} />
            ) : (
              <ColorCell
                {...props}
                text={colorText(props.value)}
                warnings={warningsAt(props.row)}
              />
            )
          }
        </Input.TableColumn>
        <Input.TableColumn name="Abnormal" type="boolean" width={14}>
          {(props) =>
            abnormalDisabled ? (
              <EmptyCell {...props} />
            ) : (
              // Each edit saves at once, so the cell is a switch.
              <Input.Switch
                value={props.value}
                onChange={props.onChange}
                variant="text"
                disabled={props.preview}
                aria-label={props["aria-label"]}
              />
            )
          }
        </Input.TableColumn>
      </Input.Table>
    </div>
  );
};

export interface EntriesProps {
  enumeration: enumeration.Key;
  /** Focuses the first Name cell on mount, or Add entry when there are no entries. */
  autoFocus?: boolean;
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
  const handleAdd = (added: enumeration.Entry[]): void => {
    let after = entries.at(-1)?.key;
    for (const entry of added) {
      form.insert(formPath.entries, entry, after);
      after = entry.key;
    }
  };
  const handleMove = (from: number, to: number): void => {
    const moved = entries[from];
    const others = entries.filter((e) => e.key !== moved.key);
    form.move(formPath.entries, moved.key, to > 0 ? others[to - 1].key : undefined);
  };
  return (
    <Table
      value={entries}
      preview={!canUpdate}
      onSet={(entry) => form.set(formPath.entries[entry.key], entry)}
      onAdd={handleAdd}
      onRemove={(i) => form.remove(formPath.entries[entries[i].key])}
      onMove={handleMove}
      autoFocus={autoFocus}
    />
  );
};
