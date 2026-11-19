// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/ContextMenu.css";

import { box, location, unique, xy } from "@synnaxlabs/x";
import {
  type ReactNode,
  type RefCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { useCombinedRefs, useResize, useSyncedRef, useWindowResize } from "@/hooks";
import { Filtered, useFilter } from "@/menu/filter";
import {
  CLOSES_ATTRIBUTE,
  CONTEXT_MENU_CLASS,
  CONTEXT_OPEN_ATTRIBUTE,
  CONTEXT_SELECTED,
  CONTEXT_TARGET,
} from "@/menu/types";
import { Portal } from "@/portal";
import { position } from "@/position";
import { Triggers } from "@/triggers";
import { FOCUSABLE } from "@/util";

interface ContextMenuState {
  visible: boolean;
  keys: string[];
  position: xy.XY;
  cursor: xy.XY;
  /** The context target the menu opened over. Null for a menu opened at a point. */
  target: HTMLElement | null;
}

/** Supported event types for triggering a context menu. */
export type ContextMenuEvent = xy.Client & {
  preventDefault: () => void;
  stopPropagation: () => void;
  target: Element;
};

export interface ContextMenuOpen {
  (e: xy.Crude | ContextMenuEvent): void;
}

/** Return value for the {@Menu.useContextMenu} hook. */
export interface UseContextMenuReturn extends ContextMenuState {
  visible: boolean;
  close: () => void;
  /**
   * Gives the focus back as a chosen item does, while the focus is in the menu: to the
   * target the menu opened over if the target can take the focus and the element that
   * had the focus when the menu opened holds it, else to that element. The menu stays
   * open. The menu surface calls it in the capture phase of an item click, so the item
   * acts with the focus already back.
   */
  returnFocus: () => void;
  open: ContextMenuOpen;
  ref: RefCallback<HTMLDivElement>;
  className: string;
}

const ESCAPE_TRIGGERS: Triggers.Trigger[] = [Triggers.ESCAPE];

const INITIAL_STATE: ContextMenuState = {
  visible: false,
  position: xy.ZERO,
  cursor: xy.ZERO,
  keys: [],
  target: null,
};

const CONTEXT_MENU_CONTAINER = CSS.BE("menu-context", "container");

const OPEN_DIALOG = `.${CSS.BE("dialog", "frame")}.${CSS.visible(true)}`;

// A click on one of these chooses it: a menu item that opens no submenu, or an element
// that closes the menu.
const CHOOSABLE = `[role="menuitem"]:not([aria-haspopup]), [${CLOSES_ATTRIBUTE}]`;

const findTarget = (target: HTMLElement): HTMLElement => {
  let candidate = target;
  while (!candidate.classList.contains(CONTEXT_TARGET)) {
    if (candidate.classList.contains(CONTEXT_MENU_CONTAINER)) return target;
    if (candidate.parentElement == null) return target;
    candidate = candidate.parentElement;
  }
  return candidate;
};

const findSelected = (target: HTMLElement): HTMLElement[] => {
  const selected: HTMLElement[] = Array.from(
    target.parentElement?.querySelectorAll(`.${CONTEXT_SELECTED}`) ?? [],
  );
  if (selected.includes(target)) return selected;
  return [target];
};

const PREFERENCES: position.LocationPreference[] = [
  { targetCorner: location.BOTTOM_RIGHT, dialogCorner: location.TOP_LEFT },
  { targetCorner: location.BOTTOM_LEFT, dialogCorner: location.TOP_RIGHT },
  { targetCorner: location.TOP_RIGHT, dialogCorner: location.BOTTOM_LEFT },
  { targetCorner: location.TOP_LEFT, dialogCorner: location.BOTTOM_RIGHT },
];

/**
 * Menu.useContextMenu extracts the logic for toggling a context menu, allowing
 * the caller to control the menu's visibility and position.
 *
 * @returns visible - Whether the menu is visible.
 * @returns close - A function to close the menu.
 * @returns open - A function to open the menu. The function accepts an XY coordinate
 * and an optional set of keys to set as the selected menu items. It's important to note
 * that these keys override the default behavior of the menu, which is explained in the
 * documentation for {@link Menu.ContextMenu}.
 */
export const useContextMenu = (): UseContextMenuReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<ContextMenuState>(INITIAL_STATE);
  // Where the focus was when the menu opened. The menu takes the focus from there, and
  // gives it back there when it closes.
  const foundRef = useRef<HTMLElement | null>(null);
  // Where a chosen item gives the focus back. A right click in WebKit focuses no
  // button. The focus stays on the body, or on a box around the button that a click can
  // focus, so a focusable target that the menu opened over takes it.
  const chosenRef = useRef<HTMLElement | null>(null);
  // Whether the focus showed its ring when the menu opened. Closing the menu with a key
  // would show the ring, so the menu gives it back as it found it.
  const ringRef = useRef(false);

  // Only the item the menu opened over holds the hover look, even when the menu acts
  // on a wider selection.
  const { target: stamped } = state;
  useEffect(() => {
    if (stamped == null) return;
    stamped.setAttribute(CONTEXT_OPEN_ATTRIBUTE, "");
    return () => stamped.removeAttribute(CONTEXT_OPEN_ATTRIBUTE);
  }, [stamped]);

  const handleOpen: ContextMenuOpen = (e) => {
    const p = xy.construct(e);
    let keys: string[] = [];
    let opened: HTMLElement | null = null;
    if (typeof e === "object" && "preventDefault" in e) {
      e.preventDefault();
      e.stopPropagation();
      const target = findTarget(e.target as HTMLElement);
      const selected = findSelected(target);
      keys = unique.unique(
        selected
          .map((el) => el.dataset.menuKey ?? el.id)
          .filter((key) => key.length > 0),
      );
      if (target.classList.contains(CONTEXT_TARGET)) opened = target;
    }
    const active = document.activeElement;
    if (active instanceof HTMLElement && menuRef.current?.contains(active) !== true) {
      foundRef.current = active;
      chosenRef.current =
        opened?.matches(FOCUSABLE) === true && active.contains(opened)
          ? opened
          : active;
      ringRef.current = active.matches(":focus-visible");
    }
    setMenuState({ visible: true, keys, position: p, cursor: p, target: opened });
  };

  const calculatePosition = () => {
    const el = menuRef.current;
    if (el == null) return;
    setMenuState((prev) => {
      if (!prev.visible) return prev;
      const { adjustedDialog } = position.position({
        container: box.construct(0, 0, window.innerWidth, window.innerHeight),
        dialog: box.construct(el),
        target: box.construct(prev.cursor, 0, 0),
        prefer: PREFERENCES,
      });
      // A menu that fits nowhere around the cursor slides into the window.
      const nextPos = xy.construct(
        Math.max(
          0,
          Math.min(
            box.left(adjustedDialog),
            window.innerWidth - box.width(adjustedDialog),
          ),
        ),
        Math.max(
          0,
          Math.min(
            box.top(adjustedDialog),
            window.innerHeight - box.height(adjustedDialog),
          ),
        ),
      );
      if (xy.equals(prev.position, nextPos)) return prev;
      return { ...prev, position: nextPos };
    });
  };

  const resizeRef = useResize(calculatePosition, { enabled: state.visible });
  // A menu that opens at a new cursor keeps its size, so no resize places it there.
  const { cursor, visible } = state;
  useLayoutEffect(() => {
    if (visible) calculatePosition();
  }, [visible, cursor]);
  const combinedRef = useCombinedRefs(menuRef, resizeRef);

  useWindowResize(calculatePosition, { enabled: state.visible });

  // The items can load after the menu opens, and an empty menu is hidden and cannot
  // take the focus, so the menu takes it when they show unless the focus has moved.
  useEffect(() => {
    const menu = menuRef.current;
    if (!state.visible || menu == null) return;
    const take = (): void => {
      if (document.activeElement === foundRef.current)
        menu.focus({ preventScroll: true });
    };
    take();
    const observer = new MutationObserver(take);
    observer.observe(menu, { childList: true });
    return () => observer.disconnect();
  }, [state.visible]);

  const giveBack = (back: HTMLElement | null): void => {
    // An item that moved the focus, such as one that opened a modal, keeps it there.
    const menu = menuRef.current;
    if (
      back?.isConnected === true &&
      menu != null &&
      Dialog.contains(menu, document.activeElement)
    )
      back.focus({ preventScroll: true, focusVisible: ringRef.current });
  };
  const returnFocus = (): void => giveBack(chosenRef.current);

  const hide = (focus: boolean): void => {
    if (focus) giveBack(foundRef.current);
    foundRef.current = null;
    chosenRef.current = null;
    setMenuState(INITIAL_STATE);
  };
  const hideMenu = (): void => hide(true);

  // A click outside leaves the focus where the click puts it.
  Dialog.useClickOutside({ ref: menuRef, onClickOutside: () => hide(false) });

  const visibleRef = useSyncedRef(state.visible);
  const handleEscape = ({ stage, stopPropagation }: Triggers.UseEvent) => {
    if (stage !== "start" || !visibleRef.current) return;
    // A menu with an open submenu leaves Escape to it, so Escape closes one level.
    if (menuRef.current?.querySelector(OPEN_DIALOG) != null) return;
    hideMenu();
    stopPropagation();
  };
  // Outranks the dialog Escape handler (priority 100) so a menu opened inside a
  // dialog closes without also closing the dialog.
  Triggers.use({
    triggers: ESCAPE_TRIGGERS,
    callback: handleEscape,
    loose: true,
    priority: 150,
  });

  return {
    ...state,
    close: hideMenu,
    returnFocus,
    open: handleOpen,
    ref: combinedRef,
    className: CONTEXT_MENU_CONTAINER,
  };
};

export interface ContextMenuMenuProps extends ContextMenuState {
  keys: string[];
}

export interface ContextMenuProps
  extends Omit<UseContextMenuReturn, "className">, Omit<Flex.BoxProps, "ref"> {
  menu?: RenderProp<ContextMenuMenuProps>;
}

// The surface of an open menu. It mounts only while the menu is open.
const Surface = ({
  ref,
  menu,
  visible,
  open,
  close,
  returnFocus,
  position,
  keys,
  className,
  cursor,
  target,
  style,
  onClick,
  onClickCapture,
  onKeyDown,
  ...rest
}: ContextMenuProps): ReactNode => {
  const filter = useFilter();
  const combinedRef = useCombinedRefs(ref, filter.ref);
  const portalTarget = Portal.useTarget();
  const menuStyle = { ...xy.css(position), ...style };
  return createPortal(
    <Flex.Box
      className={CSS.cls(CONTEXT_MENU_CLASS, CSS.bordered(), className)}
      ref={combinedRef}
      role="menu"
      tabIndex={-1}
      style={menuStyle}
      // The focus goes back before a chosen item acts, so a view or modal that the item
      // opens takes that element as its opener. The menu closes only after the item
      // acts, since React can render between the capture and bubble phases.
      onClickCapture={(e) => {
        if (e.target instanceof Element && e.target.closest(CHOOSABLE) != null)
          returnFocus();
        onClickCapture?.(e);
      }}
      onClick={(e) => {
        close();
        onClick?.(e);
      }}
      gap="tiny"
      onKeyDown={(e) => {
        onKeyDown?.(e);
        filter.onKeyDown(e);
      }}
      {...rest}
      // React bubbles events out of the portal to the owner's ancestors, where an outer
      // context menu would open on top of this one.
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
    >
      <Filtered filter={filter}>
        {menu?.({ keys, visible, position, cursor, target })}
      </Filtered>
    </Flex.Box>,
    portalTarget ?? document.body,
  );
};

const Internal = (props: ContextMenuProps): ReactNode | null =>
  props.visible ? <Surface {...props} /> : null;

/**
 * Menu.ContextMenu wraps a set of children with a context menu. When the user
 * right clicks within wrapped area, the provided menu will be shown.
 * Menu.ContextMenu should be used in conjunction with the Menu.useContextMenu
 * hook.
 *
 * The rendered menu is provided with a set of keys that identify the context
 * target elements. A target's key is its `data-menu-key` attribute when present,
 * otherwise its HTML id. Set `data-menu-key` when the element's id is reserved for
 * another purpose (e.g. a tab whose id encodes ARIA linking). The first target is
 * evaluated by traversing the parents of the element that was right clicked until
 * an element with the class "lyra-context__target" is found. If no such element is
 * found, the right clicked element itself is used as the target. If this target has
 * the class "lyra-context--selected", then subsequent targets are found by querying
 * all siblings of the first target that have the "lyra-context--selected" class.
 * Otherwise, the only key is the first target.
 *
 * While the menu is visible, the target the menu opened over (not the wider
 * selection) carries the `data-context-menu-open` attribute so styles can keep
 * it highlighted.
 *
 * @example <caption>Example DOM structure</caption>
 *   <div id="lyra-menu-context__container">
 *    <div className="lyra-context__target" id="1">
 *      <span>
 *        <h2>I was right clicked!</h2>
 *      </span>
 *    </div>
 *    <div className="lyra-context__target lyra-context--selected" id="2">
 *    <div className="lyra-context__target" id="3">
 *   </div>
 *
 * In the above example, the keys provided to the menu would be ["1"].
 *
 * If the <div> element with id="1" had a className of "lyra-context__target
 * lyra-context--selected" instead, the keys provided would be ["1", "2"].
 *
 * The target resolution logic is ideal for both single and multi-select
 * scenarios, such as lists that have several selected rows that should be acted
 * upon together.
 *
 * @param props - Props for the component. Expects all return values from the
 * useContextMenu hook. All non-hook and unlisted props will be spread to the
 * underlying div component acting as the root element.
 * @param props.menu - The menu to show when the user right clicks.
 */
export const ContextMenu = ({
  menu,
  children,
  ...rest
}: ContextMenuProps): ReactNode => (
  <>
    <Internal menu={menu} {...rest} />
    {children}
  </>
);
