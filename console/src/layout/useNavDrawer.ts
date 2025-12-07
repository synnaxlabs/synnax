// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  type Icon,
  type Nav,
  type Triggers,
  useDebouncedCallback,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useSelectNavDrawer } from "@/layout/selectors";
import {
  type NavDrawerLocation,
  resizeNavDrawer,
  setNavDrawerVisible,
  startNavHover,
  stopNavHover,
} from "@/layout/slice";

export interface NavMenuItem {
  key: string;
  icon: Icon.ReactElement;
  tooltip: string;
  trigger: Triggers.Trigger;
  useVisible?: () => boolean;
}

export interface NavDrawerItem extends Nav.DrawerItem, NavMenuItem {}

export interface UseNavDrawerReturn {
  activeItem: NavDrawerItem | undefined;
  menuItems: NavMenuItem[];
  onSelect: (item: string) => void;
  onCollapse: () => void;
  onResize: (size: number) => void;
  onStartHover: (item: string) => void;
  onStopHover: () => void;
  hover: boolean;
}

export const useNavDrawer = (
  location: NavDrawerLocation,
  items: NavDrawerItem[],
): UseNavDrawerReturn => {
  const windowKey = useSelectWindowKey() as string;
  const state = useSelectNavDrawer(location);
  const dispatch = useDispatch();
  const onResize = useDebouncedCallback(
    (size) => {
      dispatch(resizeNavDrawer({ windowKey, location, size }));
    },
    100,
    [dispatch, windowKey],
  );

  let activeItem: NavDrawerItem | undefined;
  if (state?.activeItem != null)
    activeItem = items.find((item) => item.key === state.activeItem);
  const menuItems = state?.menuItems
    .map((key) => items.find((item) => item.key === key))
    .filter((item) => item != null);

  if (activeItem != null) activeItem.initialSize = state?.size;

  const onSelect = useCallback(
    (key: string) => dispatch(setNavDrawerVisible({ windowKey, key })),
    [dispatch, windowKey],
  );

  const hoverRef = useSyncedRef(state?.hover ?? false);

  const onCollapse = useCallback(() => {
    if (hoverRef.current) dispatch(stopNavHover({ windowKey, location }));
    else
      dispatch(
        setNavDrawerVisible({ windowKey, key: undefined, value: false, location }),
      );
  }, [dispatch, windowKey, location, hoverRef]);

  const onStartHover = useCallback(
    (key: string) => dispatch(startNavHover({ windowKey, location, key })),
    [dispatch, windowKey, location],
  );

  const onStopHover = useCallback(
    () => dispatch(stopNavHover({ windowKey, location })),
    [dispatch, windowKey, location],
  );

  return {
    activeItem,
    menuItems: menuItems ?? [],
    onSelect,
    onCollapse,
    onResize,
    hover: state?.hover ?? false,
    onStartHover,
    onStopHover,
  };
};
