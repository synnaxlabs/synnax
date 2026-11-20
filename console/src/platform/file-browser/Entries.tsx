// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useCombinedRefs } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import {
  type KeyboardEvent,
  type ReactElement,
  type Ref,
  useEffect,
  useLayoutEffect,
  useRef,
} from "react";

import { CSS } from "@/platform/css";
import type { Entry } from "@/platform/file-browser/list";
import { Validity } from "@/platform/secret/validity";

/** An entry keyed by its path, as the list holds it. */
export interface KeyedEntry extends Entry {
  key: string;
}

/** The files that the dialog can pick. */
export type TypeFilter = "certificates" | "all";

/** @returns whether the entry is a file that the type filter lets the user pick. */
export const pickable = (entry: Entry, type: TypeFilter): boolean =>
  entry.kind === "file" && (type === "all" || entry.certificate != null);

type SortKey = "name" | "expires" | "size" | "modified";

/** The column that orders the list, and its direction. */
export interface Sort {
  key: SortKey;
  ascending: boolean;
}

const COLLATOR = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });

const sortValue = (entry: Entry, key: Exclude<SortKey, "name">): number => {
  if (key === "size") return entry.size ?? -1;
  if (key === "modified") return Number(entry.modified?.valueOf() ?? 0n);
  return Number(entry.certificate?.notAfter.valueOf() ?? 0n);
};

/** @returns the entries with folders first, each part in the order of the sort. */
export const sortEntries = (entries: Entry[], { key, ascending }: Sort): Entry[] => {
  const dir = ascending ? 1 : -1;
  return [...entries].sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === "directory" ? -1 : 1;
    const byName = COLLATOR.compare(a.name, b.name);
    if (key === "name") return dir * byName;
    const d = sortValue(a, key) - sortValue(b, key);
    return d === 0 ? byName : dir * d;
  });
};

interface Column extends List.Column {
  key: SortKey | null;
  numeric?: boolean;
}

/** The columns of the list, for {@link List.Columns}. */
export const COLUMNS: Column[] = [
  { key: "name", name: "Name", width: "minmax(26rem, 1.5fr)" },
  { key: null, name: "Certificate", width: "minmax(16rem, 1fr)" },
  { key: "expires", name: "Expires", width: "20rem" },
  { key: "size", name: "Size", width: "10rem", numeric: true },
  { key: "modified", name: "Modified", width: "14rem", numeric: true },
];

const NUMERIC_CLASS = CSS.BE("file-browser", "numeric");

const CELLS_CLASS = CSS.BE("file-browser", "cells");

export interface HeadProps {
  sort: Sort;
  onSort: (sort: Sort) => void;
}

/** The headings of the columns. A click on a heading sorts by its column. */
export const Head = ({ sort, onSort }: HeadProps): ReactElement => (
  <div className={CSS.BE("file-browser", "head")}>
    <List.Cells className={CELLS_CLASS}>
      {COLUMNS.map(({ key, name, numeric = false }) => {
        if (key == null)
          return (
            <List.Cell key={name}>
              <span className={CSS.BE("file-browser", "heading-text")}>{name}</span>
            </List.Cell>
          );
        const active = sort.key === key;
        const Caret = sort.ascending ? Icon.Caret.Up : Icon.Caret.Down;
        const caret = active && <Caret className={CSS.BE("file-browser", "caret")} />;
        return (
          <List.Cell key={name} className={CSS.cls(numeric && NUMERIC_CLASS)}>
            <Text.Text
              el="button"
              type="button"
              level="small"
              className={CSS.cls(
                CSS.BE("file-browser", "heading-text"),
                active && CSS.M("active"),
              )}
              onClick={() =>
                onSort({ key, ascending: active ? !sort.ascending : true })
              }
            >
              {numeric && caret}
              {name}
              {!numeric && caret}
            </Text.Text>
          </List.Cell>
        );
      })}
    </List.Cells>
  </div>
);

/**
 * @returns the most specific name in a subject in RFC 2253 form: its CN, else its O,
 * else its OU.
 */
const subjectName = (subject: string): string => {
  const values = new Map<string, string>();
  let attr = "";
  const end = (): void => {
    const i = attr.indexOf("=");
    const type = attr.slice(0, i);
    if (i > 0 && !values.has(type)) values.set(type, attr.slice(i + 1));
    attr = "";
  };
  for (let i = 0; i < subject.length; i++) {
    const c = subject[i];
    if (c === "\\") attr += subject[++i] ?? "";
    else if (c === "," || c === "+") end();
    else attr += c;
  }
  end();
  return values.get("CN") ?? values.get("O") ?? values.get("OU") ?? subject;
};

const DATE = new Intl.DateTimeFormat(undefined, {
  month: "short",
  day: "numeric",
  year: "numeric",
});

const DAY = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" });

const formatModified = (modified: TimeStamp | undefined, now: TimeStamp): string => {
  if (modified == null) return "";
  const date = modified.date();
  return date.getFullYear() === now.date().getFullYear()
    ? DAY.format(date)
    : DATE.format(date);
};

const SIZE_UNITS = ["kB", "MB", "GB", "TB"];

// One decimal below 10, whole numbers above.
const roundSize = (value: number): number =>
  value < 10 ? Math.round(value * 10) / 10 : Math.round(value);

const formatSize = (bytes: number | undefined): string => {
  if (bytes == null) return "";
  if (bytes < 1000) return `${bytes} B`;
  let value = bytes / 1000;
  let unit = 0;
  while (roundSize(value) >= 1000 && unit < SIZE_UNITS.length - 1) {
    value /= 1000;
    unit++;
  }
  const rounded = roundSize(value);
  return `${rounded < 10 ? rounded.toFixed(1) : rounded} ${SIZE_UNITS[unit]}`;
};

interface ExpiryProps {
  details: certificate.Details;
  now: TimeStamp;
}

const Expiry = ({ details, now }: ExpiryProps): ReactElement | null => {
  const validity = Validity.describeDetails(details, now);
  if (validity == null) return null;
  let text = DATE.format(details.notAfter.date());
  if (now.afterEq(details.notAfter)) text = "Expired";
  else if (now.before(details.notBefore)) text = "Not yet valid";
  return (
    <Badge.Badge status={validity.status} className={CSS.BE("file-browser", "badge")}>
      {text}
    </Badge.Badge>
  );
};

interface ItemProps extends List.ItemProps<string> {
  now: TimeStamp;
  type: TypeFilter;
  onOpen: (entry: Entry) => void;
  /** Called when the pointer enters an entry. */
  onPoint: (entry: Entry) => void;
}

const Item = ({
  now,
  type,
  onOpen,
  onPoint,
  ...rest
}: ItemProps): ReactElement | null => {
  const entry = List.useItem<string, KeyedEntry>(rest.itemKey);
  if (entry == null) return null;
  const { certificate } = entry;
  let EntryIcon: Icon.FC = Icon.File;
  if (entry.kind === "directory") EntryIcon = Icon.Folder;
  else if (certificate != null) EntryIcon = Icon.Certificate;
  return (
    <Select.Item
      {...rest}
      className={CSS.cls(
        CSS.BE("file-browser", "entry"),
        entry.kind === "file" && !pickable(entry, type) && CSS.M("dim"),
      )}
      textValue={entry.name}
      onDoubleClick={() => onOpen(entry)}
      onMouseEnter={() => onPoint(entry)}
    >
      <List.Cells className={CELLS_CLASS}>
        <List.Cell>
          <Flex.Box
            x
            align="center"
            gap="medium"
            className={CSS.BE("file-browser", "name")}
          >
            <EntryIcon
              className={CSS.cls(
                CSS.BE("file-browser", "icon"),
                certificate != null && CSS.M("certificate"),
              )}
            />
            <Text.Text overflow="fade">{entry.name}</Text.Text>
          </Flex.Box>
        </List.Cell>
        <List.Cell>
          <Text.Text level="small" color={9} overflow="fade">
            {certificate == null ? "" : subjectName(certificate.subject)}
          </Text.Text>
        </List.Cell>
        <List.Cell>
          {certificate != null && <Expiry details={certificate} now={now} />}
        </List.Cell>
        <List.Cell className={NUMERIC_CLASS}>
          <Text.Text level="small" color={9}>
            {entry.kind === "file" ? formatSize(entry.size) : ""}
          </Text.Text>
        </List.Cell>
        <List.Cell className={NUMERIC_CLASS}>
          <Text.Text level="small" color={9}>
            {formatModified(entry.modified, now)}
          </Text.Text>
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

export interface EntriesProps extends Pick<
  ItemProps,
  "now" | "type" | "onOpen" | "onPoint"
> {
  /** The folder of the entries. Each new folder scrolls to its selected entry. */
  folder: string | null;
  selected: string | null;
  ref?: Ref<HTMLDivElement>;
  /** Selects the entry with the path. */
  onSelect: (path: string) => void;
  /** Opens the parent folder on Backspace. */
  onUp: () => void;
  /** Opens the path field with a typed / or ~. */
  onEditPath: (text: string) => void;
}

// A pause this long starts a new name for the keys typed to jump to an entry.
const TYPE_AHEAD_RESET = TimeSpan.milliseconds(900);

/**
 * The entries of a folder. Enter double-clicks the selected entry, and typed letters
 * select the first entry whose name starts with them. A state that takes the place of
 * the list keeps the focus in its holder, so keys still reach the dialog.
 */
export const Entries = ({
  folder,
  selected,
  ref: propsRef,
  onSelect,
  onUp,
  onEditPath,
  ...rest
}: EntriesProps): ReactElement => {
  const keyboard = Select.useKeyboard();
  const { getData, getItem, scrollToIndex } = List.useUtilContext<string, KeyedEntry>();
  // A new folder opens at its top. The selected entry, if any, then scrolls into view.
  useEffect(() => scrollToIndex(0), [folder]);
  useEffect(() => {
    const index = selected == null ? -1 : getData().indexOf(selected);
    if (index >= 0) scrollToIndex(index);
  }, [folder, selected]);
  const ref = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(ref, propsRef);
  const typed = useRef({ text: "", at: TimeStamp.ZERO });
  useLayoutEffect(() => {
    const el = ref.current;
    const holder = el?.parentElement;
    if (holder != null && document.activeElement === holder) el?.focus();
    return () => {
      if (el?.contains(document.activeElement)) holder?.focus();
    };
  }, []);
  const typeAhead = (key: string): void => {
    const now = TimeStamp.now();
    const t = typed.current;
    t.text = t.at.span(now).greaterThan(TYPE_AHEAD_RESET) ? key : `${t.text}${key}`;
    t.at = now;
    const name = t.text.toLowerCase();
    const keys = getData();
    const i = keys.findIndex(
      (k) => getItem?.(k)?.name.toLowerCase().startsWith(name) === true,
    );
    if (i < 0) return;
    onSelect(keys[i]);
    scrollToIndex(i);
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    // AltGr reports Ctrl and Alt on Windows, and still types a character.
    if ((e.ctrlKey || e.metaKey || e.altKey) && !e.getModifierState("AltGraph"))
      keyboard.onKeyDown(e);
    else if (e.key === "/" || e.key === "~") {
      e.preventDefault();
      onEditPath(e.key);
    } else if (e.key === "Backspace") {
      e.preventDefault();
      onUp();
    } else if (e.key.length === 1 && e.key !== " ") {
      e.preventDefault();
      typeAhead(e.key);
    } else keyboard.onKeyDown(e);
  };
  return (
    <List.Scroll
      ref={combinedRef}
      role="listbox"
      aria-label="Files"
      tabIndex={0}
      className={CSS.BE("file-browser", "scroll")}
      onKeyDown={handleKeyDown}
      onKeyUp={keyboard.onKeyUp}
      onBlur={keyboard.onBlur}
    >
      <List.Items<string>>
        {({ key, ...p }) => <Item key={key} {...p} {...rest} />}
      </List.Items>
    </List.Scroll>
  );
};
