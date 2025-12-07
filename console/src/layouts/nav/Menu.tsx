// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { CSS as PCSS, Menu as PMenu, Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { selectActiveMosaicTabState } from "@/layout/selectors";
import { setNavDrawerVisible, toggleNavHover } from "@/layout/slice";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { type RootState } from "@/store";

interface MenuItemProps {
  item: Layout.NavMenuItem;
  isActive: boolean;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
}

const MenuItem = ({
  item,
  isActive,
  onStartHover,
  onStopHover,
}: MenuItemProps): ReactElement | null => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const windowKey = useSelectWindowKey();

  const isVisible = item.useVisible?.() ?? true;
  const isVisibleRef = useSyncedRef(isVisible);

  const { key, icon, trigger } = item;

  // Build triggers regardless of visibility: single press + double press.
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
          e.stage !== "start" ||
          windowKey == null ||
          (e.prevTriggers.length > 0 && e.prevTriggers[0].length > 1)
        )
          return;
        const state = store.getState();
        const { blurred } = selectActiveMosaicTabState(state, windowKey);
        if (blurred) return;

        const isDouble = e.triggers.some((t) => t.length === 2);
        if (isDouble) dispatch(setNavDrawerVisible({ windowKey, key, value: true }));
        else dispatch(toggleNavHover({ windowKey, key }));
      },
      [dispatch, windowKey, key, store, isVisibleRef],
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
  location: Layout.NavDrawerLocation;
}

export const Menu = ({ location, ...rest }: MenuProps): ReactElement => {
  const { onSelect, menuItems, activeItem, onStartHover, onStopHover } =
    Layout.useNavDrawer(location, DRAWER_ITEMS);

  return (
    <PMenu.Menu {...rest} onChange={onSelect}>
      {menuItems.map((item) => (
        <MenuItem
          key={item.key}
          item={item}
          isActive={activeItem?.key === item.key}
          onStartHover={onStartHover}
          onStopHover={onStopHover}
        />
      ))}
    </PMenu.Menu>
  );
};
