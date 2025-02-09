// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/pluto";
import { compare, type Key, type Keyed } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { DataProvider } from "@/list/Data";
import { InfiniteProvider } from "@/list/Infinite";
import { useMemoCompare } from "@/memo";

export interface ListProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends PropsWithChildren<unknown> {
  data?: E[];
  emptyContent?: ReactElement;
  omit?: K[];
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
  omit,
}: ListProps<K, E>): ReactElement => {
  const omittedData = useMemoCompare(
    () => (omit != null ? data?.filter((e) => !omit.includes(e.key)) : data),
    ([prevOmit, prevData], [omit, data]) => {
      let omitsEqual: boolean;
      if (prevOmit != null && omit != null)
        omitsEqual = compare.unorderedPrimitiveArrays(prevOmit, omit) === compare.EQUAL;
      else omitsEqual = prevOmit == omit;
      return prevData === data && omitsEqual;
    },
    [omit, data] as [K[] | undefined, E[] | undefined],
  );
  const newEmptyContent =
    typeof emptyContent === "string" ? (
      <Text.Text level="p">{emptyContent}</Text.Text>
    ) : (
      emptyContent
    );
  return (
    <InfiniteProvider>
      <DataProvider<K, E> data={omittedData} emptyContent={newEmptyContent}>
        {children}
      </DataProvider>
    </InfiniteProvider>
  );
};
