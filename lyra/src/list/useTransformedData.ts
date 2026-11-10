// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { useInitializerRef } from "@/hooks";
import { createGetItem, type FrameProps } from "@/list/Frame";

export interface UseTransformedDataParams<
  K extends record.Key,
  E extends record.Keyed<K>,
  TK extends record.Key,
  TE extends record.Keyed<TK>,
> {
  /** The list data source to transform. */
  source: Pick<FrameProps<K, E>, "data" | "getItem" | "subscribe">;
  /** Returns the key of the item that a source key transforms to. */
  toKey: (key: K) => TK;
  /** Returns the source key of a transformed key, or null when it has none. */
  fromKey: (key: TK) => K | null;
  /** Transforms a source item. It runs once for each source item. */
  transform: (item: E) => TE;
}

/**
 * Presents a list data source as a list of another item type. A source item keeps the
 * same transformed item until the source replaces it, so rows that read items through
 * `subscribe` stay stable.
 */
export const useTransformedData = <
  K extends record.Key,
  E extends record.Keyed<K>,
  TK extends record.Key,
  TE extends record.Keyed<TK>,
>({
  source,
  toKey,
  fromKey,
  transform,
}: UseTransformedDataParams<K, E, TK, TE>): Required<
  Pick<FrameProps<TK, TE>, "data" | "getItem" | "subscribe">
> => {
  const transformed = useInitializerRef(() => new WeakMap<E, TE>());
  const getOne = (key: TK): TE | undefined => {
    const sourceKey = fromKey(key);
    if (sourceKey == null) return undefined;
    const item = source.getItem?.(sourceKey);
    if (item == null) return undefined;
    const cached = transformed.current.get(item);
    if (cached != null) return cached;
    const next = transform(item);
    transformed.current.set(item, next);
    return next;
  };
  const getItem = createGetItem<TK, TE>(getOne, (keys) =>
    keys.map(getOne).filter((item) => item != null),
  );
  const subscribe = (callback: () => void, key: TK) => {
    const sourceKey = fromKey(key);
    if (sourceKey == null || source.subscribe == null) return () => {};
    return source.subscribe(callback, sourceKey);
  };
  return { data: source.data.map(toKey), getItem, subscribe };
};
