// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context, type Icon, type Nav, type Triggers } from "@synnaxlabs/pluto";

export interface MenuItem {
  key: string;
  icon: Icon.ReactElement;
  tooltip: string;
  trigger: Triggers.Trigger;
  useVisible?: () => boolean;
}

export interface Item extends Nav.DrawerItem, MenuItem {}

export interface ContextValue {
  left: Item[];
  bottom: Item;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Nav.Item.Context",
  providerName: "Nav.Item.Provider",
});
export { Context };

export const useLeft = (): Item[] => useContext("useLeft").left;

export const useFindLeft = (key: string): Item => {
  const r = useContext("useLeft").left.find((i) => i.key == key);
  if (r == null) throw new Error(`no nav item for ${key}`);
  return r;
};

export const useBottom = (): Item => useContext("useLeft").bottom;
