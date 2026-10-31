// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { array, direction, type location, xy } from "@synnaxlabs/x";
import { type MouseEvent as ReactMouseEvent, type ReactElement, useRef } from "react";

import { CSS } from "@/platform/css";
import { type Toolbar, useVisibleKeys } from "@/platform/nav/toolbar";

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

  const triggers =
    trigger == null || trigger.length === 0 ? [] : [trigger, [trigger[0], trigger[0]]];

  Triggers.use({
    triggers,
    loose: false,
    callback: (e: Triggers.UseEvent) => {
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
  });

  const cancelPending = () => {
    if (timeoutRef.current == null) return;
    clearTimeout(timeoutRef.current);
    timeoutRef.current = null;
  };

  const onMouseEnter = (e: ReactMouseEvent) => {
    const alongAxis = direction.swap(bar);
    const crossAxis = direction.construct(bar);
    timeoutRef.current = setTimeout(() => {
      timeoutRef.current = null;
      onStartHover();
      positionRef.current = xy.construct(e);
      const controller = new AbortController();
      window.addEventListener(
        "mousemove",
        (e: MouseEvent) => {
          const delta = xy.translation(xy.construct(e), positionRef.current);
          if (Math.abs(delta[alongAxis]) > 75 && Math.abs(delta[crossAxis]) < 30) {
            onStopHover();
            controller.abort();
          }
        },
        { signal: controller.signal },
      );
    }, 350);
  };

  return { onClick: cancelPending, onMouseEnter, onMouseLeave: cancelPending };
};

interface MenuItemProps {
  item: Toolbar;
  visible: boolean;
  selected: boolean;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
  onToggle: (key: string) => void;
  onPin: (key: string) => void;
}

const MenuItem = ({
  item,
  visible,
  selected,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
}: MenuItemProps): ReactElement | null => {
  const { key, icon, tooltip, trigger, badge } = item;
  const itemProps = useItem({
    trigger,
    visible,
    onStartHover: () => onStartHover(key),
    onStopHover,
    onToggle: () => onToggle(key),
    onPin: () => onPin(key),
  });

  if (!visible) return null;

  return (
    <PMenu.Item
      key={key}
      className={CSS.cls(CSS.BE("main-nav", "item"), PCSS.selected(selected))}
      {...itemProps}
      itemKey={key}
      size="medium"
      triggerIndicator={trigger}
      // The visible content is an icon plus a trigger-key hint, neither of
      // which names the item, so the tooltip doubles as the accessible name.
      aria-label={tooltip}
    >
      {icon}
      {badge}
    </PMenu.Item>
  );
};

export interface MenuProps
  extends
    Omit<PMenu.MenuProps, "children" | "onChange">,
    Pick<MenuItemProps, "onStartHover" | "onStopHover" | "onToggle" | "onPin"> {
  /** A module-level constant, since each toolbar's visibility hook runs in order. */
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
}: MenuProps): ReactElement => {
  const toolbars = array.toArray(items);
  const visible = useVisibleKeys(toolbars);
  return (
    <PMenu.Menu {...rest} onChange={onSelect}>
      {toolbars.map((item) => (
        <MenuItem
          key={item.key}
          item={item}
          visible={visible.has(item.key)}
          selected={selected === item.key}
          onStartHover={onStartHover}
          onStopHover={onStopHover}
          onToggle={onToggle}
          onPin={onPin}
        />
      ))}
    </PMenu.Menu>
  );
};
