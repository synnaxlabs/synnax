// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/filter.css";

import {
  isValidElement,
  type KeyboardEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  type RefObject,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Icon } from "@/icon";
// The input imports the menu, so the menu imports the search directly.
import { Search, type SearchProps } from "@/input/Search";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { ACTIVATION_KEYS, isMoveKey, moveIndex } from "@/util";

const [FilterContext, useFilterContext] = context.create<string>({
  defaultValue: "",
  displayName: "Menu.FilterContext",
});

export { FilterContext };

const MENU_ITEMS = '[role="menuitem"]:not([aria-disabled="true"])';

const SURFACES = `[role="menu"], .${CSS.B("menu-dialog")}`;

const HIGHLIGHTED = "data-highlighted";

/** @returns the strings and numbers in a node, read through element children. */
export const nodeText = (node: ReactNode): string => {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(nodeText).join("");
  if (isValidElement<PropsWithChildren>(node)) return nodeText(node.props.children);
  return "";
};

/**
 * @returns whether an item with the given text shows under the filter of its menu.
 * @param text - The item's text.
 */
export const useMatches = (text: string): boolean => {
  const term = useFilterContext();
  return term === "" || text.toLowerCase().includes(term.toLowerCase());
};

/** What a menu surface needs to filter its items as the user types. */
export interface Filter {
  /** The typed filter, or null while the menu shows no search. */
  term: string | null;
  /** Attach to the menu surface. */
  ref: RefObject<HTMLDivElement | null>;
  /** Handles the filter and move keys. Pass it to the menu surface. */
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
  /** Props for the search. */
  field: Required<Pick<SearchProps, "ref" | "onChange" | "onKeyDown">>;
}

/**
 * Lets a menu surface filter its items as the user types. Call it in a part that
 * mounts only while the menu is open. The surface takes the focus when it mounts,
 * unless the focus is already in it. The first printable key opens a search at the
 * top. While the search has the focus, the first match is lit and Enter picks it.
 * Escape closes the search, then the menu. Keys from another field or a submenu keep
 * their own meaning.
 */
export const useFilter = (): Filter => {
  const [term, setTerm] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);
  const fieldRef = useRef<HTMLInputElement>(null);
  const editedRef = useRef(false);
  useEffect(() => {
    const surface = ref.current;
    if (surface != null && !surface.contains(document.activeElement))
      surface.focus({ preventScroll: true });
  }, []);
  // A key typed on an item goes on in the search, after the text already there.
  useLayoutEffect(() => {
    const field = fieldRef.current;
    if (!editedRef.current || field == null) return;
    editedRef.current = false;
    field.focus({ preventScroll: true });
    field.setSelectionRange(field.value.length, field.value.length);
  }, [term]);
  useLayoutEffect(() => {
    const surface = ref.current;
    if (surface == null) return;
    surface.querySelector(`[${HIGHLIGHTED}]`)?.removeAttribute(HIGHLIGHTED);
    if (term != null) surface.querySelector(MENU_ITEMS)?.setAttribute(HIGHLIGHTED, "");
  }, [term]);
  const close = (): void => {
    ref.current?.focus({ preventScroll: true });
    setTerm(null);
  };
  const edit = (e: KeyboardEvent<HTMLElement>, next: string | null): void => {
    e.preventDefault();
    e.stopPropagation();
    if (next == null) return close();
    editedRef.current = true;
    setTerm(next);
  };
  const onKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if (e.defaultPrevented || e.altKey || e.ctrlKey || e.metaKey) return;
    const surface = e.currentTarget;
    const target = e.target as HTMLElement;
    if (target !== surface && target.closest(SURFACES) !== surface) return;
    if (target !== surface && !target.matches(MENU_ITEMS)) return;
    if (term != null && (e.key === "Escape" || e.key === "Backspace"))
      return edit(e, e.key === "Escape" ? null : term.slice(0, -1));
    if (e.key.length === 1 && (e.key !== " " || (term ?? "") !== ""))
      return edit(e, (term ?? "") + e.key);
    if (e.shiftKey) return;
    // The menu keeps these keys from a list behind it that answers window keys.
    if (ACTIVATION_KEYS.includes(e.key)) return e.stopPropagation();
    const key = Triggers.eventKey(e);
    if (!isMoveKey(key)) return;
    e.stopPropagation();
    const items = Array.from(surface.querySelectorAll<HTMLElement>(MENU_ITEMS));
    if (items.length === 0) return;
    e.preventDefault();
    const index = moveIndex(key, items.indexOf(target), items.length, e.repeat);
    if (index != null) items[index].focus();
  };
  const onFieldKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    const surface = ref.current;
    if (surface == null) return;
    if (e.key === "Escape") {
      e.stopPropagation();
      return close();
    }
    // Enter picks the lit item: the one under the pointer, else the first match.
    if (e.key === "Enter") {
      e.stopPropagation();
      const item =
        surface.querySelector<HTMLElement>(`${MENU_ITEMS}:hover`) ??
        surface.querySelector<HTMLElement>(MENU_ITEMS);
      return item?.click();
    }
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    e.preventDefault();
    e.stopPropagation();
    const items = Array.from(surface.querySelectorAll<HTMLElement>(MENU_ITEMS));
    const index = moveIndex(e.key, -1, items.length, e.repeat);
    if (index != null) items[index]?.focus();
  };
  return {
    term,
    ref,
    onKeyDown,
    field: { ref: fieldRef, onChange: setTerm, onKeyDown: onFieldKeyDown },
  };
};

/** Props for {@link Filtered}. */
export interface FilteredProps extends PropsWithChildren {
  filter: Filter;
}

/**
 * The items of a menu under its filter. While the user types, a search at the top
 * holds the filter, and a muted line says when no item matches.
 */
export const Filtered = ({
  filter: { term, field },
  children,
}: FilteredProps): ReactElement => (
  <FilterContext value={term ?? ""}>
    {term != null && (
      <Search
        {...field}
        value={term}
        className={CSS.B("menu-filter")}
        flush
        startContent={<Icon.Search />}
        placeholder="Search..."
        full="x"
      />
    )}
    {children}
    {term != null && term !== "" && (
      <Text.Text color={8} className={CSS.BE("menu-filter", "empty")}>
        No matches
      </Text.Text>
    )}
  </FilterContext>
);
