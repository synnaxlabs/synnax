// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Items.css";

import { type record } from "@synnaxlabs/x";
import { type ReactNode, useMemo } from "react";

import { memo } from "@/component/memo";
import { CSS } from "@/css";
import { useData } from "@/list/Frame";
import { type ItemRenderProp } from "@/list/Item";
import { ItemsContext, useElementsContext, useScrollContext } from "@/list/scope";

/** Props for {@link Items}. */
export interface ItemsProps<K extends record.Key = record.Key> {
  /** Renders one item. It is called once per visible key. */
  children: ItemRenderProp<K>;
  /** Rendered in place of the items when the list is empty. */
  emptyContent?: ReactNode;
}

const BaseItems = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  children,
  emptyContent,
}: ItemsProps<K>): ReactNode => {
  const scrollRef = useScrollContext("List.Items");
  const { ref, itemsRef, getItems, getTotalSize, data, sentinelRef } = useData<K, E>();
  const { setElement } = useElementsContext("List.Items");
  if (scrollRef !== ref)
    throw new Error("List.Items must be inside the List.Scroll of its own List.Frame");
  const totalSize = getTotalSize();
  const virtualizerStyle = useMemo(() => ({ minHeight: totalSize }), [totalSize]);
  if (data.length === 0) return emptyContent;
  return (
    <div
      ref={itemsRef}
      className={CSS.BE("list", "virtualizer")}
      style={virtualizerStyle}
    >
      <ItemsContext value={setElement}>
        {getItems().map(({ key, index, translate }) =>
          children({ key, index, itemKey: key, translate }),
        )}
      </ItemsContext>
      {sentinelRef != null && (
        <div
          ref={sentinelRef}
          className={CSS.BE("list", "sentinel")}
          aria-hidden="true"
        />
      )}
    </div>
  );
};

/**
 * Renders the visible items of a {@link Frame} inside the enclosing {@link Scroll},
 * handling virtualization, and shows `emptyContent` when there are none.
 *
 * @throws if no {@link Scroll} of the same {@link Frame} encloses it.
 */
export const Items = memo(BaseItems);
