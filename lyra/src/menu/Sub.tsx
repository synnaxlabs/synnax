// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Sub.css";

import {
  Children,
  isValidElement,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { useCombinedRefs } from "@/hooks";
import { Icon } from "@/icon";
import { nodeText, useMatches } from "@/menu/filter";
import { useContext } from "@/menu/Menu";
import { SubContext, useSubContext, useSubmenusContext } from "@/menu/submenus";
import { type position } from "@/position";

const MENU_ITEM = '[role="menuitem"]';

// The content opens beside the item, on the side that has room.
const LOCATION: position.LocationPreference = {
  targetCorner: { y: "top" },
  dialogCorner: { y: "top" },
};

// A submenu matches the filter of its menu on the text of its trigger.
const triggerText = (children: ReactNode): string => {
  const trigger = Children.toArray(children).find(
    (child): child is ReactElement<SubTriggerProps> =>
      isValidElement(child) && child.type === SubTrigger,
  );
  if (trigger == null) return "";
  return trigger.props.textValue ?? nodeText(trigger.props.children);
};

/** Props for {@link Sub}. */
export interface SubProps extends PropsWithChildren {
  /** Names the submenu among the items of its menu. */
  itemKey: string;
}

/**
 * A menu item that opens a submenu beside its menu. Its children are a {@link
 * SubTrigger} and the content: a {@link Dialog}, or a selection dialog on the menu
 * surface. A menu opens one submenu at a time. ArrowLeft and Escape close one level. A
 * click inside the content keeps a context menu open unless it picks an item.
 *
 * @example
 * <Menu.Menu>
 *   <Menu.Sub itemKey="labels">
 *     <Menu.SubTrigger><Icon.Label />Labels</Menu.SubTrigger>
 *     <Menu.Dialog>{items}</Menu.Dialog>
 *   </Menu.Sub>
 * </Menu.Menu>
 */
export const Sub = ({ itemKey, children }: SubProps): ReactElement | null => {
  const { open, swap, register, enter, setOpen, frame } = useSubmenusContext();
  const matches = useMatches(triggerText(children));
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
  if (!matches) return null;
  return (
    <SubContext value={ctx}>
      <Dialog.Frame
        variant="floating"
        location={LOCATION}
        {...frame(itemKey)}
        className={CSS.B("menu-sub")}
        onPointerMove={handlePointerMove}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
      >
        {children}
      </Dialog.Frame>
    </SubContext>
  );
};

/** Props for {@link SubTrigger}. */
export interface SubTriggerProps extends Omit<
  Button.ButtonProps,
  "variant" | "propagateClick"
> {
  /**
   * The submenu's name as plain text, for children that hold no text. The filter of a
   * menu matches it. Defaults to the text in the children.
   */
  textValue?: string;
}

/**
 * The item of a {@link Sub}. A click, Enter, Space, or ArrowRight opens the submenu at
 * once, and so does a rest of the pointer. It stays lit while the submenu is open.
 */
export const SubTrigger = ({
  ref,
  textValue: _,
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
