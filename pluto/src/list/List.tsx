// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { type Keyed, type Key } from "@synnaxlabs/x";

import { DataProvider } from "@/list/Data";
import { InfiniteProvider } from "@/list/Infinite";

export interface ListProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends PropsWithChildren<unknown> {
  data?: E[];
  emptyContent?: ReactElement;
}

/**
 * The main component for building a List. By itself, it does not render any HTML, and
 * should be used in conjunction with its sub-components (List.'X') to build a list
 * component to fit your needs.
 *
 * @param props - The props for the List component.
 * @param props.data - The data to be displayed in the list. The values of the object in
 * each entry of the array must satisfy the {@link RenderableValue} interface i.e. they
 * must be a primitive type or implement a 'toString' method.
 * @param props.children - Sub-components of the List component to add additional functionality.
 *
 */
export const List = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  children,
  data,
  emptyContent,
}: ListProps<K, E>): ReactElement => {
  return (
    <InfiniteProvider>
      <DataProvider<K, E> data={data} emptyContent={emptyContent}>
        {children}
      </DataProvider>
    </InfiniteProvider>
  );
};
