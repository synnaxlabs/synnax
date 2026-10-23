// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/IconPicker.css";

import { type color, type dimensions, grid, type xy } from "@synnaxlabs/x";
import Fuse from "fuse.js";
import {
  type KeyboardEvent,
  type ReactElement,
  useMemo,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Search } from "@/input/Search";
import { Text } from "@/text";

export interface IconPickerProps {
  /** The key of the picked {@link Icon.Pickable}. */
  value?: string;
  /** Called with the key of the icon the user picks. */
  onChange: (key: string) => void;
  /** Draws every icon in this color, through {@link Icon.tint}. */
  color?: color.Crude;
}

const COLUMNS = 10;

type Move = (dims: dimensions.Dimensions, from: xy.XY) => xy.XY | null;

/** The cell each navigation key moves focus to. */
const MOVES: Partial<Record<string, Move>> = {
  ArrowRight: (dims, from) => grid.next(dims, from, 1),
  ArrowLeft: (dims, from) => grid.next(dims, from, -1),
  ArrowDown: (dims, from) => grid.move(dims, from, { x: 0, y: 1 }),
  ArrowUp: (dims, from) => grid.move(dims, from, { x: 0, y: -1 }),
  Home: () => ({ x: 0, y: 0 }),
  End: (dims) => ({ x: dims.width - 1, y: dims.height - 1 }),
};

/**
 * A search field over a grid of the {@link Icon.PICKABLE} icons. The arrow keys move
 * between the field and the grid, Enter in the field picks the best match, and Escape in
 * the grid returns to the field.
 */
export const IconPicker = ({
  value,
  onChange,
  color,
}: IconPickerProps): ReactElement => {
  const [term, setTerm] = useState("");
  const [focused, setFocused] = useState<string>();
  const fuse = useMemo(
    () => new Fuse(Icon.PICKABLE, { keys: ["name", "tags"], threshold: 0.3 }),
    [],
  );
  const icons = term === "" ? Icon.PICKABLE : fuse.search(term).map(({ item }) => item);
  const searchRef = useRef<HTMLInputElement>(null);
  const cellRefs = useRef(new Map<string, HTMLElement>());
  const indexOf = (key?: string): number => icons.findIndex((i) => i.key === key);
  const focusedIndex = indexOf(focused);
  const active = focusedIndex >= 0 ? focusedIndex : Math.max(indexOf(value), 0);
  const focusCell = (index: number): void =>
    cellRefs.current.get(icons[index].key)?.focus();

  const handleSearchKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (icons.length === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      focusCell(active);
    } else if (e.key === "Enter") onChange(icons[0].key);
  };

  const handleGridKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if (e.key === "Escape") {
      e.stopPropagation();
      searchRef.current?.focus();
      return;
    }
    const move = MOVES[e.key];
    if (move == null) return;
    e.preventDefault();
    const dims = { width: COLUMNS, height: Math.ceil(icons.length / COLUMNS) };
    const next = move(dims, { x: active % COLUMNS, y: Math.floor(active / COLUMNS) });
    if (next == null) {
      if (e.key === "ArrowUp") searchRef.current?.focus();
      return;
    }
    focusCell(Math.min(next.y * COLUMNS + next.x, icons.length - 1));
  };

  return (
    <Flex.Box y gap="medium" className={CSS.B("icon-picker")}>
      <Search
        ref={searchRef}
        value={term}
        onChange={setTerm}
        onKeyDown={handleSearchKeyDown}
        size="small"
        full="x"
        placeholder="Search icons"
        aria-label="Search icons"
      />
      {icons.length === 0 ? (
        <Text.Text level="small" color={9}>
          No icons match
        </Text.Text>
      ) : (
        <div
          role="listbox"
          aria-label="Icons"
          className={CSS.BE("icon-picker", "grid")}
          style={{ [CSS.variable("icon-picker", "columns")]: COLUMNS }}
          onKeyDown={handleGridKeyDown}
        >
          {icons.map(({ key, name, icon: Glyph }, i) => (
            <Button.Button
              key={key}
              ref={(el: HTMLElement | null) => {
                if (el == null) cellRefs.current.delete(key);
                else cellRefs.current.set(key, el);
              }}
              role="option"
              aria-selected={key === value}
              aria-label={name}
              tooltip={name}
              variant="text"
              size="small"
              tabIndex={i === active ? 0 : -1}
              className={CSS.cls(CSS.selected(key === value))}
              onFocus={() => setFocused(key)}
              onClick={() => onChange(key)}
            >
              <Glyph {...Icon.tint(color)} />
            </Button.Button>
          ))}
        </div>
      )}
    </Flex.Box>
  );
};
