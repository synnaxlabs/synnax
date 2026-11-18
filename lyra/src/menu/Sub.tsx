// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Sub.css";

import { type xy } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { useCombinedRefs } from "@/hooks";
import { Icon } from "@/icon";
import {
  FilterContext,
  isFilterKey,
  PathContext,
  useMatches,
  usePath,
} from "@/menu/filter";
import { useContext } from "@/menu/Menu";
import { SubContext, useSubContext, useSubmenusContext } from "@/menu/submenus";
import { type position } from "@/position";

const MENU_ITEM = '[role="menuitem"]';
const LIST = '[role="listbox"]';

// The content opens beside the item, on the side that has room. It laps 4 px over the
// menu edge: padding 6 + border 1 - frame offset 3.
const LOCATION: position.LocationPreference = {
  targetCorner: { y: "top" },
  dialogCorner: { y: "top" },
};

// The list area of the content sits level with the item that opens it. It starts below
// the search of a selection, or at the padding of a menu.
const alignOffset = (dialog: HTMLElement): xy.XY => {
  const list = dialog.querySelector(LIST);
  const y =
    list == null
      ? dialog.clientTop + parseFloat(getComputedStyle(dialog).paddingTop)
      : list.getBoundingClientRect().top - dialog.getBoundingClientRect().top;
  return { x: 0, y };
};

/** Props for {@link Sub}. */
export interface SubProps extends PropsWithChildren {
  /** Names the submenu among the items of its menu. */
  itemKey: string;
  /** The submenu's name as plain text, which the filter of its menu matches. */
  textValue: string;
}

/**
 * A menu item that opens a submenu beside its menu. Its children are a {@link
 * SubTrigger} and the content: a {@link Dialog}, or a selection dialog on the menu
 * surface. A menu opens one submenu at a time. ArrowLeft and Escape close one level. A
 * click inside the content keeps a context menu open unless it picks an item. While its
 * menu filters by text that its name does not match, the items of a {@link Dialog} show
 * in the menu, with the name of the submenu before each label. Text typed outside a
 * field of the submenu closes it and goes to the filter of the menu.
 *
 * @example
 * <Menu.Menu>
 *   <Menu.Sub itemKey="labels" textValue="Labels">
 *     <Menu.SubTrigger><Icon.Label />Labels</Menu.SubTrigger>
 *     <Menu.Dialog>{items}</Menu.Dialog>
 *   </Menu.Sub>
 * </Menu.Menu>
 */
export const Sub = ({
  itemKey,
  textValue,
  children,
}: SubProps): ReactElement | null => {
  const { open, swap, register, enter, setOpen, frame } = useSubmenusContext();
  const path = usePath();
  const matches = useMatches(textValue);
  const trigger = useRef<HTMLElement | null>(null);
  const content = useRef<HTMLElement | null>(null);
  const visible = open === itemKey;
  useEffect(() => register(itemKey, content), [register, itemKey]);
  // The keys go to the list that holds the focus, so the content takes it.
  useEffect(() => {
    const el = content.current;
    if (!visible || el == null || el.contains(document.activeElement)) return;
    (el.querySelector<HTMLElement>(MENU_ITEM) ?? el).focus({ preventScroll: true });
  }, [visible]);
  const isTrigger = (target: EventTarget): boolean =>
    target instanceof Node && trigger.current?.contains(target) === true;
  const handlePointerMove = (e: PointerEvent<HTMLElement>): void => {
    if (!isTrigger(e.target)) enter(e);
  };
  const handleClick = (e: MouseEvent<HTMLElement>): void => {
    if (!(e.target instanceof Element) || e.target.closest(MENU_ITEM) == null)
      e.stopPropagation();
  };
  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    // The text goes on to the filter of the menu, which the open submenu would cover.
    if (visible && isFilterKey(e)) return setOpen(null);
    if (
      e.key !== "ArrowLeft" ||
      e.defaultPrevented ||
      e.altKey ||
      e.ctrlKey ||
      e.metaKey ||
      e.shiftKey ||
      !visible ||
      isTrigger(e.target)
    )
      return;
    if (e.target instanceof HTMLInputElement && e.target.value !== "") return;
    e.preventDefault();
    setOpen(null);
  };
  const ctx = { itemKey, trigger, content, swap: swap && visible };
  if (!matches)
    return <PathContext value={[...path, textValue]}>{children}</PathContext>;
  return (
    <SubContext value={ctx}>
      <Dialog.Frame
        variant="floating"
        location={LOCATION}
        alignOffset={alignOffset}
        {...frame(itemKey)}
        className={CSS.B("menu-sub")}
        onPointerMove={handlePointerMove}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
      >
        {/* The filter of the menu matches the submenu, never its content. */}
        <FilterContext value="">
          <PathContext value={[]}>{children}</PathContext>
        </FilterContext>
      </Dialog.Frame>
    </SubContext>
  );
};

/** Props for {@link SubTrigger}. */
export interface SubTriggerProps extends Omit<
  Button.ButtonProps,
  "variant" | "propagateClick"
> {}

/**
 * The item of a {@link Sub}. A click, Enter, Space, or ArrowRight opens the submenu at
 * once, and so does a rest of the pointer. It stays lit while the submenu is open. It
 * is absent while the items of the submenu show in its menu.
 */
export const SubTrigger = (props: SubTriggerProps): ReactElement | null =>
  usePath().length > 0 ? null : <Trigger {...props} />;

const Trigger = ({
  ref,
  className,
  children,
  onClick,
  onKeyDown,
  onPointerMove,
  onPointerLeave,
  ...rest
}: SubTriggerProps): ReactElement => {
  const { level = "p", gap } = useContext();
  const { open, aiming, hovered, setOpen, item } = useSubmenusContext();
  const { itemKey, trigger } = useSubContext("Menu.SubTrigger");
  const { triggerRef } = Dialog.useContext();
  const combinedRef = useCombinedRefs(ref, triggerRef, trigger);
  const handlers = item(itemKey);
  const expanded = open === itemKey;
  // Only one item is lit at a time, so it goes dark while the pointer rests on another.
  const lit = expanded && (aiming || hovered == null || hovered === itemKey);
  return (
    <Button.Button
      ref={combinedRef}
      level={level}
      gap={gap}
      variant="text"
      overflow="nowrap"
      role="menuitem"
      aria-haspopup="dialog"
      aria-expanded={expanded}
      className={CSS.cls(
        CSS.B("menu-item"),
        CSS.BE("menu-sub", "trigger"),
        lit && CSS.BEM("menu-sub", "trigger", "lit"),
        aiming && CSS.BM("menu-item", "aiming"),
        className,
      )}
      onClick={(e) => {
        onClick?.(e);
        setOpen(itemKey);
      }}
      onKeyDown={(e) => {
        onKeyDown?.(e);
        if (e.key !== "ArrowRight" || e.defaultPrevented) return;
        e.preventDefault();
        setOpen(itemKey);
      }}
      onPointerMove={(e) => {
        onPointerMove?.(e);
        handlers.onPointerMove(e);
      }}
      onPointerLeave={(e) => {
        onPointerLeave?.(e);
        handlers.onPointerLeave(e);
      }}
      {...rest}
    >
      {children}
      <Icon.Caret.Right color={8} className={CSS.BE("menu-sub", "caret")} />
    </Button.Button>
  );
};
