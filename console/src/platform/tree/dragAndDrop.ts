// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, type Synnax as Client } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Status } from "@synnaxlabs/lyra/status";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { type DragEvent, type RefObject } from "react";

import { findAncestors, findContainedKeys } from "@/platform/tree/ancestry";
import { type Item } from "@/platform/tree/item";
import { type ResultNodes } from "@/platform/tree/search";
import { type Store } from "@/platform/tree/store";
import { type GetResource } from "@/platform/tree/types";

export interface UseDragAndDropProps {
  client: Client;
  store: Store;
  root: resource.ID;
  results?: ResultNodes;
  resolveItem: (type: resource.Type) => Item;
  selectedRef: RefObject<string[]>;
  contract: (...keys: string[]) => void;
  getResource: GetResource;
}

export interface DragAndDrop {
  /** Moves the dropped items under the item of the key. Returns the items it moves. */
  onDrop: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onDragStart: (key: string) => void;
  onDragEnd: (e: DragEvent) => void;
}

/** Drags the selection, or the item alone when it is not selected, between items. */
export const useDragAndDrop = ({
  client,
  store,
  root,
  results,
  resolveItem,
  selectedRef,
  contract,
  getResource,
}: UseDragAndDropProps): DragAndDrop => {
  const handleError = Status.useErrorHandler();
  const { startDrag, onDragEnd } = Haul.useDrag({ type: Base.HAUL_TYPE });

  const onDrop = (key: string, { source, items }: Haul.OnDropProps): Haul.Item[] => {
    const dropped = Base.filterHaulItems(items);
    if (dropped.length === 0 || source.type !== Base.HAUL_TYPE) return [];
    const destination = store.get(key);
    if (destination == null) return [];
    const { canDrop, drop } = resolveItem(destination.id.type);
    if (drop == null) return [];
    // An item inside another dragged item travels with it. Moving it explicitly would
    // re-parent it onto the destination as a sibling of its own parent.
    const contained = findContainedKeys(
      store,
      dropped.map(({ key }) => key),
      results?.ancestors,
    );
    const haulMoved = dropped.filter(({ key }) => !contained.has(key));
    if (haulMoved.length === 0) return [];
    const moved = haulMoved.map(({ key }) => ({
      id: resource.parseID(key),
      ancestors: findAncestors(store, key, root, results?.paths),
    }));
    const ancestors = findAncestors(store, key, root, results?.paths);
    if (!canDrop({ source, items, root, destination, ancestors, moved })) return [];
    contract(...haulMoved.map(({ key }) => key));
    handleError(
      async () => await drop({ client, destination, items: moved }),
      "Failed to move resources",
    );
    return haulMoved;
  };

  const onDragStart = (key: string): void => {
    if (selectedRef.current.includes(key)) {
      const selectedItems = getResource(selectedRef.current).flatMap((res) => [
        Base.createHaulItem(resource.idToString(res.id)),
        ...resolveItem(res.id.type).haulItems(res),
      ]);
      return startDrag(selectedItems);
    }
    const haulItems = resolveItem(resource.parseID(key).type).haulItems(
      getResource(key),
    );
    startDrag([Base.createHaulItem(key), ...haulItems]);
  };

  return { onDrop, onDragStart, onDragEnd };
};
