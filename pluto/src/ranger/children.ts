// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, type ranger, type Synnax as Client } from "@synnaxlabs/client";
import { useMemoArray, useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { useEffect, useState } from "react";

import { Synnax } from "@/synnax";

/** How many children of a range each read adds. */
const PAGE_SIZE = 50;

const RAN: ranger.Order = { field: "time_range_start", descending: false };

const FAILED = "Failed to retrieve child ranges";

const NO_PARENTS: ranger.Key[] = [];

export interface UseChildrenParams {
  /** The ranges whose children to read. */
  parents: ranger.Key[];
  /** The order of the children. Defaults to the order in which they ran. */
  order?: ranger.Order;
}

export interface Children {
  /** The keys of the children read so far, in order. */
  keys: ranger.Key[];
  /** Whether the keys hold every child of the range. */
  complete: boolean;
}

export interface UseChildrenReturn {
  /** The children of each parent. A parent is absent until its first read lands. */
  children: Map<ranger.Key, Children>;
  /** Reads the next page of the children of the parent. */
  more: (parent: ranger.Key) => void;
}

/** The live query of the children of each parent, made once per parent. */
interface Lists {
  client: Client | null;
  order: ranger.Order;
  byParent: Map<ranger.Key, query.List<ranger.Range, unknown>>;
}

/**
 * Reads the children of each parent one page at a time, and keeps them live. A failed
 * read adds an error status.
 */
export const useChildren = ({
  parents,
  order = RAN,
}: UseChildrenParams): UseChildrenReturn => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const stableOrder = useMemoDeepEqual(order);
  const keys = useMemoArray(parents);
  const [lists, setLists] = useState<Lists>(() => ({
    client,
    order: stableOrder,
    byParent: new Map(),
  }));
  if (lists.client !== client || lists.order !== stableOrder)
    setLists({ client, order: stableOrder, byParent: new Map() });
  const read = lists.client == null ? NO_PARENTS : keys;
  const listOf = (key: ranger.Key): query.List<ranger.Range, unknown> => {
    let list = lists.byParent.get(key);
    if (list != null) return list;
    if (lists.client == null) throw new Error("useChildren reads with no client");
    list = lists.client.ranges.watch({
      where: { parents: [key] },
      order: lists.order,
      limit: PAGE_SIZE,
    });
    lists.byParent.set(key, list);
    return list;
  };
  const windows = Store.useItems({
    keys: read as ranger.Key[],
    subscribe: (listener, key) => listOf(key).onChange(listener),
    getItem: (keys) => keys.map((key) => listOf(key).getCached()),
  });
  useEffect(() => {
    read.forEach((key) => {
      const list = listOf(key);
      if (list.getCached() == null)
        handleError(async () => await list.retrieve(), FAILED);
    });
  }, [lists, read]);
  const children = new Map<ranger.Key, Children>();
  read.forEach((key, i) => {
    const window = windows[i];
    if (window == null) return;
    children.set(key, {
      keys: window.records.map((r) => r.key),
      complete: window.end == null,
    });
  });
  return {
    children,
    more: (key) => handleError(async () => await listOf(key).more(PAGE_SIZE), FAILED),
  };
};
