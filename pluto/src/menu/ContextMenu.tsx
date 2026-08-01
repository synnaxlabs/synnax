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
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import {
  useClickOutside,
  useCombinedRefs,
  useResize,
  useSyncedRef,
  useWindowResize,
} from "@/hooks";
import {
  CONTEXT_MENU_CLASS,
  CONTEXT_OPEN_ATTRIBUTE,
  CONTEXT_SELECTED,
  CONTEXT_TARGET,
} from "@/menu/types";
import { Triggers } from "@/triggers";

interface ContextMenuState {
  visible: boolean;
  keys: string[];
  position: xy.XY;
  cursor: xy.XY;
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
  open: ContextMenuOpen;
  ref: RefCallback<HTMLDivElement>;
  className: string;
}

const ESCAPE_TRIGGERS: Triggers.Trigger[] = [["Escape"]];

const INITIAL_STATE: ContextMenuState = {
  visible: false,
  position: xy.ZERO,
  cursor: xy.ZERO,
  keys: [],
};

const CONTEXT_MENU_CONTAINER = CSS.BE("menu-context", "container");

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

const PREFERENCES: Dialog.LocationPreference[] = [
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
 * @returns open - A function to open the menu. The function accepts an XY coordinate and
 * an optional set of keys to set as the selected menu items. It's important to note
 * that these keys override the default behavior of the menu, which is explained in
 * the documentation for {@link Menu.ContextMenu}.
 */
export const useContextMenu = (): UseContextMenuReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<ContextMenuState>(INITIAL_STATE);
  const targetRef = useRef<HTMLElement | null>(null);

  const clearTarget = useCallback(() => {
    targetRef.current?.removeAttribute(CONTEXT_OPEN_ATTRIBUTE);
    targetRef.current = null;
  }, []);

  // Clear the stamped target if the hook unmounts while a menu is open.
  useEffect(() => clearTarget, [clearTarget]);

  const handleOpen: ContextMenuOpen = useCallback(
    (e) => {
      const p = xy.construct(e);
      clearTarget();
      let keys: string[] = [];
      if (typeof e === "object" && "preventDefault" in e) {
        e.preventDefault();
        // Prevent parent context menus from opening.
        e.stopPropagation();
        const target = findTarget(e.target as HTMLElement);
        const selected = findSelected(target);
        keys = unique.unique(
          selected
            .map((el) => el.dataset.menuKey ?? el.id)
            .filter((key) => key.length > 0),
        );
        // Only the row the menu physically opened over holds the hover look, even
        // when the menu acts on a wider selection.
        if (target.classList.contains(CONTEXT_TARGET)) {
          targetRef.current = target;
          target.setAttribute(CONTEXT_OPEN_ATTRIBUTE, "");
        }
      }
      setMenuState({ visible: true, keys, position: p, cursor: p });
    },
    [clearTarget],
  );

  const calculatePosition = useCallback(() => {
    const el = menuRef.current;
    if (el == null) return;
    setMenuState((prev) => {
      if (!prev.visible) return prev;
      const { adjustedDialog } = Dialog.position({
        container: box.construct(0, 0, window.innerWidth, window.innerHeight),
        dialog: box.construct(el),
        target: box.construct(prev.cursor, 0, 0),
        prefer: PREFERENCES,
      });
      const nextPos = box.topLeft(adjustedDialog);
      if (xy.equals(prev.position, nextPos)) return prev;
      return { ...prev, position: nextPos };
    });
  }, []);

  const resizeRef = useResize(calculatePosition, { enabled: state.visible });
  const combinedRef = useCombinedRefs(menuRef, resizeRef);

  useWindowResize(calculatePosition, { enabled: state.visible });

  const hideMenu = useCallback((): void => {
    clearTarget();
    setMenuState(INITIAL_STATE);
  }, [clearTarget]);

  useClickOutside({ ref: menuRef, onClickOutside: hideMenu });

  const visibleRef = useSyncedRef(state.visible);
  const handleEscape = useCallback(
    ({ stage, stopPropagation }: Triggers.UseEvent) => {
      if (stage !== "start" || !visibleRef.current) return;
      hideMenu();
      stopPropagation();
    },
    [hideMenu],
  );
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

const Internal = ({
  ref,
  menu,
  visible,
  open,
  close,
  position,
  keys,
  className,
  cursor,
  style,
  onClick,
  ...rest
}: ContextMenuProps): ReactNode | null => {
  const menuStyle = useMemo(
    () => ({ ...xy.css(position), ...style }),
    [position, style],
  );
  if (!visible) return null;
  return createPortal(
    <Flex.Box
      className={CSS(CONTEXT_MENU_CLASS, CSS.bordered(), className)}
      ref={ref}
      style={menuStyle}
      onClick={(e) => {
        close();
        onClick?.(e);
      }}
      gap="tiny"
      {...rest}
    >
      {menu?.({ keys, visible, position, cursor })}
    </Flex.Box>,
    document.body,
  );
};

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
 * an element with the class "pluto-context__target" is found. If no such element is
 * found, the right clicked element itself is used as the target. If this target has
 * the class "pluto-context--selected", then subsequent targets are found by querying
 * all siblings of the first target that have the "pluto-context--selected" class.
 * Otherwise, the only key is the first target.
 *
 * While the menu is visible, the target the menu opened over (not the wider
 * selection) carries the `data-context-menu-open` attribute so styles can keep
 * it highlighted.
 *
 * @example <caption>Example DOM structure</caption>
 *   <div id="pluto-menu-context__container">
 *    <div className="pluto-context__target" id="1">
 *      <span>
 *        <h2>I was right clicked!</h2>
 *      </span>
 *    </div>
 *    <div className="pluto-context__target pluto-context--selected" id="2">
 *    <div className="pluto-context__target" id="3">
 *   </div>
 *
 * In the above example, the keys provided to the menu would be ["1"].
 *
 * If the <div> element with id="1" had a className of "pluto-context__target
 * pluto-context--selected" instead, the keys provided would be ["1", "2"].
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
