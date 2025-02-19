// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { type Icon, type Nav, useDebouncedCallback } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelectNavDrawer } from "@/layout/selectors";
import {
  type NavDrawerLocation,
  resizeNavDrawer,
  setNavDrawerVisible,
} from "@/layout/slice";

export interface NavMenuItem {
  key: string;
  icon: Icon.Element;
  tooltip: string;
}

export interface NavDrawerItem extends Nav.DrawerItem, NavMenuItem {}

export interface UseNavDrawerReturn {
  activeItem: NavDrawerItem | undefined;
  menuItems: NavMenuItem[];
  onSelect: (item: string) => void;
  onResize: (size: number) => void;
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
  if (state == null)
    return {
      activeItem: undefined,
      menuItems: [],
      onSelect: () => {},
      onResize: () => {},
    };
  let activeItem: NavDrawerItem | undefined;
  if (state.activeItem != null)
    activeItem = items.find((item) => item.key === state.activeItem);
  const menuItems = items.filter((item) => state.menuItems.includes(item.key));

  if (activeItem != null) activeItem.initialSize = state.size;

  return {
    activeItem,
    menuItems,
    onSelect: (key: string) => dispatch(setNavDrawerVisible({ windowKey, key })),
    onResize,
  };
};
