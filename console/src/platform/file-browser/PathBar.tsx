// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import {
  type KeyboardEvent,
  type MouseEvent,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { sortEntries } from "@/platform/file-browser/Entries";
import { type Entry, type Listing } from "@/platform/file-browser/list";
import {
  dirnameOf,
  expandHome,
  normalize,
  segmentsOf,
  separatorOf,
  withSeparator,
} from "@/platform/file-browser/path";

const MAX_SUGGESTIONS = 8;

const NAME_SORT = { key: "name", ascending: true } as const;

const commonPrefix = (names: string[]): string => {
  if (names.length === 0) return "";
  const [first] = names;
  let end = first.length;
  for (const name of names.slice(1)) {
    let i = 0;
    while (
      i < end &&
      i < name.length &&
      name[i].toLowerCase() === first[i].toLowerCase()
    )
      i++;
    end = i;
  }
  return first.slice(0, end);
};

interface SegmentsProps {
  path: string;
  onGo: (path: string) => void;
  onEdit: (text: string) => void;
}

const Segments = ({ path, onGo, onEdit }: SegmentsProps): ReactElement => {
  const segments = segmentsOf(path);
  // A root is a place in the sidebar, so the bar leaves it out below it.
  const shown = segments.length > 1 ? segments.slice(1) : segments;
  return (
    <Flex.Box
      x
      align="center"
      empty
      bordered
      rounded
      role="group"
      aria-label="Path"
      className={CSS.BE("file-browser", "path")}
      onClick={() => onEdit(withSeparator(path))}
    >
      <Icon.Computer className={CSS.BE("file-browser", "path-icon")} />
      {shown.map((s, i) => (
        <Flex.Box
          x
          align="center"
          empty
          key={s.path}
          className={CSS.BE("file-browser", "segment")}
        >
          {i > 0 && (
            <Icon.Caret.Right className={CSS.BE("file-browser", "separator")} />
          )}
          <Button.Button
            variant="text"
            size="tiny"
            textColor={i === shown.length - 1 ? 11 : 9}
            className={CSS.BE("file-browser", "segment-button")}
            onClick={(e: MouseEvent) => {
              e.stopPropagation();
              onGo(s.path);
            }}
          >
            <Text.Text el="span" overflow="fade">
              {s.label}
            </Text.Text>
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};

interface FieldProps extends Pick<PathBarProps, "home" | "listings" | "onPrefetch"> {
  initial: string;
  onSubmit: (text: string) => void;
  onCancel: () => void;
  onBlur: () => void;
}

const Field = ({
  initial,
  home,
  listings,
  onPrefetch,
  onSubmit,
  onCancel,
  onBlur,
}: FieldProps): ReactElement => {
  const [text, setText] = useState(initial);
  const [active, setActive] = useState(-1);
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => ref.current?.focus(), []);
  const expanded = expandHome(text, home);
  const dir = normalize(dirnameOf(expanded));
  const sep = separatorOf(expanded === "" ? "/" : expanded);
  const cut = expanded.lastIndexOf(sep);
  const prefix = expanded.slice(cut + 1);
  // The folder of the typed text lists in the background, for its suggestions.
  useEffect(() => onPrefetch(dir), [onPrefetch, dir]);
  const listing = listings.get(dir);
  const typed = prefix.toLowerCase();
  // Hidden entries show only for a name that starts with a dot, as in a shell.
  const matched =
    listing == null
      ? []
      : sortEntries(
          listing.entries.filter(
            (e) =>
              (!e.name.startsWith(".") || typed.startsWith(".")) &&
              e.name.toLowerCase().startsWith(typed) &&
              (e.name !== prefix || e.kind === "directory"),
          ),
          NAME_SORT,
        );
  const suggestions = matched.slice(0, MAX_SUGGESTIONS);
  const edit = (next: string): void => {
    setActive(-1);
    setText(next);
  };
  const complete = (entry: Entry): void =>
    edit(entry.kind === "directory" ? withSeparator(entry.path) : entry.path);
  const choose = (entry: Entry): void => {
    if (entry.kind === "file") onSubmit(entry.path);
    else complete(entry);
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (suggestions.length === 0) return;
      const step = e.key === "ArrowDown" ? 1 : -1;
      // With nothing highlighted, ArrowUp starts at the last suggestion.
      setActive((a) =>
        a < 0 && step < 0
          ? suggestions.length - 1
          : (a + step + suggestions.length) % suggestions.length,
      );
    } else if (e.key === "Tab") {
      // Tab completes the highlighted or only match, else the prefix that all share.
      e.preventDefault();
      let pick: Entry | undefined = suggestions[active];
      if (pick == null && suggestions.length === 1) [pick] = suggestions;
      const common = commonPrefix(matched.map((m) => m.name));
      if (pick != null) complete(pick);
      else if (common.length > prefix.length)
        edit(`${expanded.slice(0, cut + 1)}${common}`);
      else if (suggestions.length > 0) setActive(0);
    } else if (e.key === "Enter") {
      e.preventDefault();
      e.stopPropagation();
      const pick = active >= 0 ? suggestions[active] : undefined;
      if (pick != null) choose(pick);
      else onSubmit(expanded);
    } else if (e.key === "Escape") {
      // The field closes first, so the dialog stays open.
      e.preventDefault();
      e.stopPropagation();
      onCancel();
    }
  };
  return (
    <div className={CSS.BE("file-browser", "field")}>
      <Input.Text
        ref={ref}
        value={text}
        onChange={edit}
        size="small"
        grow
        aria-label="Path"
        startContent={<Icon.Computer />}
        placeholder="Go to a folder, such as /etc/ssl/certs or ~/certs"
        spellCheck={false}
        onKeyDown={handleKeyDown}
        onBlur={onBlur}
      />
      {suggestions.length > 0 && (
        <Flex.Box
          y
          empty
          bordered
          rounded="small"
          role="listbox"
          aria-label="Suggestions"
          className={CSS.BE("file-browser", "suggestions")}
        >
          {suggestions.map((s, i) => (
            <Button.Button
              key={s.path}
              variant="text"
              size="small"
              justify="start"
              role="option"
              aria-selected={i === active}
              className={CSS.cls(
                CSS.BE("file-browser", "suggestion"),
                i === active && CSS.M("active"),
              )}
              onMouseDown={(e: MouseEvent) => {
                // The field keeps the focus, so it stays open.
                e.preventDefault();
                choose(s);
              }}
            >
              {s.kind === "directory" ? <Icon.Folder /> : <Icon.File />}
              <Text.Text el="span" overflow="fade">
                {s.kind === "directory" ? withSeparator(s.name) : s.name}
              </Text.Text>
            </Button.Button>
          ))}
        </Flex.Box>
      )}
    </div>
  );
};

export interface PathBarProps {
  /** The open folder. */
  path: string;
  /** The home folder of the Driver, which a typed ~ stands for. */
  home: string | undefined;
  /** The listings that the suggestions come from. */
  listings: ReadonlyMap<string, Listing>;
  /** The text the field opened with, or null while the bar shows the path. */
  editing: string | null;
  /** Opens the field with the text, or closes it with null. */
  onEdit: (text: string | null) => void;
  /** Closes the field on Escape. */
  onCancel: () => void;
  /** Lists a folder in the background. */
  onPrefetch: (path: string) => void;
  /** Opens a folder on the way to the open one. */
  onGo: (path: string) => void;
  /** Goes to a typed path. */
  onSubmit: (text: string) => void;
}

/**
 * The path of the open folder. Its folders open on a click, and a click elsewhere on it
 * turns it into a field where the user types a path.
 */
export const PathBar = ({
  path,
  editing,
  onEdit,
  onGo,
  ...rest
}: PathBarProps): ReactElement =>
  editing == null ? (
    <Segments path={path} onGo={onGo} onEdit={onEdit} />
  ) : (
    <Field initial={editing} onBlur={() => onEdit(null)} {...rest} />
  );
