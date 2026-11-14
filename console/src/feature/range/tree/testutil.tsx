// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { createRenderCounter } from "@synnaxlabs/lyra/testutil";
import { type ComponentType, type ReactElement } from "react";

import { FlatItem, Item } from "@/feature/range/tree/Item";
import { type LayoutProps } from "@/feature/range/tree/Layout";
import { NodeProvider } from "@/feature/range/tree/tree";

export interface CountedItemsProps extends Pick<
  LayoutProps,
  "nodes" | "expanded" | "onExpand" | "more"
> {
  layout: "tree" | "list";
}

export interface ItemCounter {
  /** Shows the ranges as the tree or list layout does, and counts each item. */
  Items: (props: CountedItemsProps) => ReactElement;
  /** The re-render counts by range key since creation or the last reset. */
  counts: Map<string, number>;
  reset: () => void;
}

/** @returns a counter of the re-renders of each range item. */
export const createItemCounter = (): ItemCounter => {
  const { Counted, counts, reset } = createRenderCounter();
  const counted =
    (Inner: ComponentType<List.ItemProps<ranger.Key>>) =>
    ({ key, ...props }: List.ItemProps<ranger.Key>): ReactElement => (
      <Counted key={key} id={props.itemKey}>
        <Inner {...props} />
      </Counted>
    );
  const items = { tree: counted(Item), list: counted(FlatItem) };
  const Items = ({ layout, ...rest }: CountedItemsProps): ReactElement => (
    <NodeProvider {...rest}>
      <List.Scroll>
        <List.Items<ranger.Key>>{items[layout]}</List.Items>
      </List.Scroll>
    </NodeProvider>
  );
  return { Items, counts, reset };
};
