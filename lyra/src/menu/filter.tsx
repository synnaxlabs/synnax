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
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { ACTIVATION_KEYS, isMoveKey, moveIndex } from "@/util";

const [FilterContext, useFilterContext] = context.create<string>({
  defaultValue: "",
  displayName: "Menu.FilterContext",
});

const MENU_ITEMS = '[role="menuitem"]:not([aria-disabled="true"])';

const SURFACES = `[role="menu"], .${CSS.B("menu-dialog")}`;

/** @returns the strings and numbers in a node, read through element children. */
export const nodeText = (node: ReactNode): string => {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(nodeText).join("");
  if (isValidElement<PropsWithChildren>(node)) return nodeText(node.props.children);
  return "";
};

/**
 * @returns whether an item with the given text shows under the filter of its menu.
 * @param text - The item's text. Defaults to the plain text of its children.
 */
export const useMatches = (text: string): boolean => {
  const term = useFilterContext();
  return term === "" || text.toLowerCase().includes(term.toLowerCase());
};

/** What a menu surface needs to filter its items as the user types. */
export interface Filter {
  /** The typed filter, empty when the menu is not filtering. */
  term: string;
  /** Handles the filter and move keys. Pass it to the menu surface. */
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
}

/**
 * Lets a menu surface filter its items as the user types. The first printable key
 * starts the filter, Backspace edits it, and Escape clears it before it closes the
 * menu. The arrow keys move the focus over the items that show. Keys from a field, or
 * from a submenu, keep their own meaning.
 *
 * @param open - Whether the menu is open. A menu that closes clears its filter.
 */
export const useFilter = (open: boolean = true): Filter => {
  const [term, setTerm] = useState("");
  const [wasOpen, setWasOpen] = useState(open);
  // Each opening starts with no filter.
  if (open !== wasOpen) {
    setWasOpen(open);
    if (!open) setTerm("");
  }
  const surfaceRef = useRef<HTMLElement | null>(null);
  // The focused item can hide, so the first item that shows takes the focus.
  useLayoutEffect(() => {
    const surface = surfaceRef.current;
    if (surface == null) return;
    const first = surface.querySelector<HTMLElement>(MENU_ITEMS);
    (first ?? surface).focus({ preventScroll: true });
  }, [term]);
  const edit = (e: KeyboardEvent<HTMLElement>, next: string): void => {
    e.preventDefault();
    e.stopPropagation();
    surfaceRef.current = e.currentTarget;
    setTerm(next);
  };
  const onKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if (e.defaultPrevented || e.altKey || e.ctrlKey || e.metaKey) return;
    const surface = e.currentTarget;
    const target = e.target as HTMLElement;
    if (target !== surface && target.closest(SURFACES) !== surface) return;
    if (target !== surface && !target.matches(MENU_ITEMS)) return;
    if (e.key === "Escape" || e.key === "Backspace") {
      if (term === "") return;
      return edit(e, e.key === "Escape" ? "" : term.slice(0, -1));
    }
    if (e.key.length === 1 && (e.key !== " " || term !== ""))
      return edit(e, term + e.key);
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
  return { term, onKeyDown };
};

/** Props for {@link Filtered}. */
export interface FilteredProps extends PropsWithChildren {
  /** The typed filter. */
  term: string;
}

/**
 * The items of a menu under its filter. While the user types, the filter shows as a
 * line at the top, and a muted line says when no item matches.
 */
export const Filtered = ({ term, children }: FilteredProps): ReactElement => (
  <FilterContext value={term}>
    {term !== "" && (
      <Text.Text overflow="nowrap" className={CSS.B("menu-filter")}>
        <Icon.Search />
        {term}
      </Text.Text>
    )}
    {children}
    {term !== "" && (
      <Text.Text color={8} className={CSS.BE("menu-filter", "empty")}>
        No matches
      </Text.Text>
    )}
  </FilterContext>
);
