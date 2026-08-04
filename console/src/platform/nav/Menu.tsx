// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS as PCSS, Menu as PMenu, Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { array, direction, type location, xy } from "@synnaxlabs/x";
import {
  type MouseEvent as ReactMouseEvent,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/platform/css";
import { type Toolbar } from "@/platform/nav/toolbar";

export interface UseItemProps {
  trigger?: Triggers.Trigger;
  /** Edge the hosting bar sits on; orients the hover-dismiss axis. */
  bar?: location.Outer;
  visible?: boolean;
  onStartHover: () => void;
  onStopHover: () => void;
  onToggle: () => void;
  onPin: () => void;
}

export interface UseItemReturn {
  onClick: () => void;
  onMouseEnter: (e: ReactMouseEvent) => void;
  onMouseLeave: () => void;
}

/**
 * Implements the interaction mechanics of a toolbar item that opens a drawer:
 * dwelling on the item starts a hover preview, moving along the bar dismisses it,
 * the trigger toggles the drawer, and a double-tap trigger pins it. Returns handlers
 * to spread onto the item element; onClick only cancels a pending hover preview, so
 * compose it with the item's own select handler.
 */
export const useItem = ({
  trigger,
  bar = "left",
  visible = true,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
}: UseItemProps): UseItemReturn => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const visibleRef = useSyncedRef(visible);

  const triggers = useMemo(() => {
    if (!trigger?.length) return [];
    return [trigger, [trigger[0], trigger[0]]];
  }, [trigger]);

  Triggers.use({
    triggers,
    loose: false,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (
          !visibleRef.current ||
          e.stage !== "start" ||
          (e.prevTriggers.length > 0 && e.prevTriggers[0].length > 1)
        )
          return;
        const isDouble = e.triggers.some((t) => t.length === 2);
        if (isDouble) onPin();
        else onToggle();
      },
      [onToggle, onPin, visibleRef],
    ),
  });

  const cancelPending = useCallback(() => {
    if (timeoutRef.current == null) return;
    clearTimeout(timeoutRef.current);
    timeoutRef.current = null;
  }, []);

  const onMouseEnter = useCallback(
    (e: ReactMouseEvent) => {
      const alongAxis = direction.swap(bar);
      const crossAxis = direction.construct(bar);
      timeoutRef.current = setTimeout(() => {
        timeoutRef.current = null;
        onStartHover();
        positionRef.current = xy.construct(e);
        const lis = (e: MouseEvent) => {
          const delta = xy.translation(xy.construct(e), positionRef.current);
          if (Math.abs(delta[alongAxis]) > 75 && Math.abs(delta[crossAxis]) < 30) {
            onStopHover();
            window.removeEventListener("mousemove", lis);
          }
        };
        window.addEventListener("mousemove", lis);
      }, 350);
    },
    [onStartHover, onStopHover, bar],
  );

  return { onClick: cancelPending, onMouseEnter, onMouseLeave: cancelPending };
};

interface MenuItemProps {
  item: Toolbar;
  selected: boolean;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
  onToggle: (key: string) => void;
  onPin: (key: string) => void;
}

const MenuItem = ({
  item,
  selected,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
}: MenuItemProps): ReactElement | null => {
  const { key, icon, trigger, useVisible } = item;
  const isVisible = useVisible?.() ?? true;
  const itemProps = useItem({
    trigger,
    visible: isVisible,
    onStartHover: useCallback(() => onStartHover(key), [onStartHover, key]),
    onStopHover,
    onToggle: useCallback(() => onToggle(key), [onToggle, key]),
    onPin: useCallback(() => onPin(key), [onPin, key]),
  });

  if (!isVisible) return null;

  return (
    <PMenu.Item
      className={CSS(CSS.BE("main-nav", "item"), PCSS.selected(selected))}
      {...itemProps}
      key={key}
      itemKey={key}
      size="medium"
      triggerIndicator={trigger}
    >
      {icon}
    </PMenu.Item>
  );
};

export interface MenuProps
  extends
    Omit<PMenu.MenuProps, "children" | "onChange">,
    Pick<MenuItemProps, "onStartHover" | "onStopHover" | "onToggle" | "onPin"> {
  items: Toolbar | Toolbar[];
  selected?: string;
  onSelect: (key: string) => void;
}

export const Menu = ({
  items,
  selected,
  onSelect,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
  ...rest
}: MenuProps): ReactElement => (
  <PMenu.Menu {...rest} onChange={onSelect}>
    {array.toArray(items).map((item) => (
      <MenuItem
        key={item.key}
        item={item}
        selected={selected === item.key}
        onStartHover={onStartHover}
        onStopHover={onStopHover}
        onToggle={onToggle}
        onPin={onPin}
      />
    ))}
  </PMenu.Menu>
);
