// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useMemo } from "react";

import { type Haul } from "@/haul";
import { useContext } from "@/mosaic/Frame";
import {
  filterTabCreateHaulItems,
  filterTabDropHaulItems,
  HAUL_DROP_TYPE,
} from "@/mosaic/haul";
import { type Tabs } from "@/tabs";

export interface UseSelectorDropPropsParams {
  /** The key of the leaf whose tab strip is being wired, as passed to its Leaf. */
  nodeKey: number;
  /**
   * The keys of the tabs currently in the leaf. Used to reject drops that would
   * remove every tab from the leaf and re-insert into it, such as dropping a leaf's
   * sole tab back onto its own strip.
   */
  tabKeys: string[];
}

export type UseSelectorDropPropsReturn = Required<
  Pick<Tabs.SelectorProps, "haulType" | "canDrop" | "onDrop">
>;

/**
 * useSelectorDropProps wires the Tabs.Selector composed inside a {@link Leaf} up
 * as the drop target for the leaf's tab strip. Spread the returned props onto the
 * Selector: strip drops then reach the Frame's onDrop and onCreate handlers with
 * location "center" and the resolved insertion index, claimed before the leaf's
 * own drop target sees them.
 */
export const useSelectorDropProps = ({
  nodeKey,
  tabKeys,
}: UseSelectorDropPropsParams): UseSelectorDropPropsReturn => {
  const { onDrop, onCreate } = useContext("Mosaic.useSelectorDropProps");
  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) => {
      if (filterTabCreateHaulItems(items).length > 0) return true;
      const dropped = filterTabDropHaulItems(items).map(({ key }) => key);
      if (dropped.length === 0) return false;
      return tabKeys.length === 0 || tabKeys.some((key) => !dropped.includes(key));
    },
    [tabKeys],
  );
  const handleDrop = useCallback(
    ({ items, index }: Tabs.SelectorOnDropParams): Haul.Item[] => {
      const dropped = filterTabDropHaulItems(items);
      if (dropped.length > 0)
        onDrop?.({
          nodeKey,
          tabKey: dropped[0].key,
          data: dropped[0].data,
          location: "center",
          index,
        });
      const created = filterTabCreateHaulItems(items);
      if (created.length > 0)
        onCreate?.({
          nodeKey,
          location: "center",
          tabKeys: created.map(({ key }) => key),
          index,
        });
      return dropped;
    },
    [nodeKey, onDrop, onCreate],
  );
  return useMemo(
    () => ({ haulType: HAUL_DROP_TYPE, canDrop, onDrop: handleDrop }),
    [canDrop, handleDrop],
  );
};
