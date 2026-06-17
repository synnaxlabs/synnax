// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { CSS as PCSS, Menu as PMenu, Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { CSS } from "@/css";
import { type NavMenuItem } from "@/nav/item";

interface MenuItemProps {
  item: NavMenuItem;
  isActive: boolean;
  enabled: boolean;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
  onToggle: (key: string) => void;
  onPin: (key: string) => void;
}

const MenuItem = ({
  item,
  isActive,
  enabled,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
}: MenuItemProps): ReactElement | null => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const isVisible = item.useVisible?.() ?? true;
  const isVisibleRef = useSyncedRef(isVisible);
  const enabledRef = useSyncedRef(enabled);

  const { key, icon, trigger } = item;

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
          !isVisibleRef.current ||
          !enabledRef.current ||
          e.stage !== "start" ||
          (e.prevTriggers.length > 0 && e.prevTriggers[0].length > 1)
        )
          return;
        const isDouble = e.triggers.some((t) => t.length === 2);
        if (isDouble) onPin(key);
        else onToggle(key);
      },
      [key, onToggle, onPin, isVisibleRef, enabledRef],
    ),
  });

  if (!isVisible) return null;

  return (
    <PMenu.Item
      className={CSS(CSS.BE("main-nav", "item"), PCSS.selected(isActive))}
      onClick={() => {
        if (timeoutRef.current != null) {
          clearTimeout(timeoutRef.current);
          timeoutRef.current = null;
        }
      }}
      onMouseEnter={(e) => {
        timeoutRef.current = setTimeout(() => {
          timeoutRef.current = null;
          onStartHover(key);
          positionRef.current = xy.construct(e);
          const lis = (e: MouseEvent) => {
            const delta = xy.translation(xy.construct(e), positionRef.current);
            if (Math.abs(delta.y) > 75 && Math.abs(delta.x) < 30) {
              onStopHover();
              window.removeEventListener("mousemove", lis);
            }
          };
          window.addEventListener("mousemove", lis);
        }, 350);
      }}
      onMouseLeave={() => {
        if (timeoutRef.current != null) {
          clearTimeout(timeoutRef.current);
          timeoutRef.current = null;
        }
      }}
      key={key}
      itemKey={key}
      size="large"
      contrast={2}
      triggerIndicator={trigger}
    >
      {icon}
    </PMenu.Item>
  );
};

export interface MenuProps extends Omit<PMenu.MenuProps, "children" | "onChange"> {
  items: NavMenuItem[];
  activeKey?: string;
  enabled?: boolean;
  onSelect: (key: string) => void;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
  onToggle: (key: string) => void;
  onPin: (key: string) => void;
}

export const Menu = ({
  items,
  activeKey,
  enabled = true,
  onSelect,
  onStartHover,
  onStopHover,
  onToggle,
  onPin,
  ...rest
}: MenuProps): ReactElement => (
  <PMenu.Menu {...rest} onChange={onSelect}>
    {items.map((item) => (
      <MenuItem
        key={item.key}
        item={item}
        isActive={activeKey === item.key}
        enabled={enabled}
        onStartHover={onStartHover}
        onStopHover={onStopHover}
        onToggle={onToggle}
        onPin={onPin}
      />
    ))}
  </PMenu.Menu>
);
