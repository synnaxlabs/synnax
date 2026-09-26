// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode, useMemo } from "react";

import { List as BaseList } from "@/list";
import { useVisibleCount } from "@/select/registry";
import { useClosed } from "@/select/scope";

/* Height the list may take, leaving the search input its share of the dialog's cap in
   Dialog.css. Rows are only ever whole, so this is a budget rather than a limit. */
const LIST_BUDGET = 220;

/**
 * useDisplayItems returns how many rows fit the dialog at the enclosing list's row
 * height. Counting rows rather than pixels gives the list a definite height, which is
 * what lets its growth animate.
 */
const useDisplayItems = (): number => {
  const itemHeight = BaseList.useItemHeight();
  return useMemo(
    () => (itemHeight == null ? 1 : Math.max(1, Math.floor(LIST_BUDGET / itemHeight))),
    [itemHeight],
  );
};

/** Props for {@link List}. */
export interface ListProps extends Omit<BaseList.ScrollProps, "itemCount"> {}

/**
 * The scroll area of a selection dialog. The fixed {@link Item}s and the {@link Items}
 * block inside it scroll together; anything outside it stays pinned. By default it
 * sizes itself to the whole rows that fit the dialog, so its growth animates.
 */
export const List = ({
  children,
  displayItems,
  animateHeight = true,
  ...rest
}: ListProps): ReactNode => {
  const closed = useClosed();
  const fixed = useVisibleCount();
  const { data } = BaseList.useData();
  const defaultDisplayItems = useDisplayItems();
  if (closed) return children;
  return (
    <BaseList.Scroll
      role="listbox"
      itemCount={data.length + fixed}
      displayItems={displayItems ?? defaultDisplayItems}
      animateHeight={animateHeight}
      {...rest}
    >
      {children}
    </BaseList.Scroll>
  );
};
