// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { DataProvider } from "./Data";
import { InfiniteProvider } from "./Infinite";

export interface ListProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends PropsWithChildren<unknown> {
  data?: E[];
  emptyContent?: ReactElement;
}

/**
 * The main component for building a List. By itself, it does not render any HTML, and
 * should be used in conjunction with its subcomponents (List.'X') to build a list
 * component to fit your needs.
 *
 * @param props - The props for the List component.
 * @param props.data - The data to be displayed in the list. The values of the object in
 * each entry of the array must satisfy the {@link RenderableValue} interface i.e. they
 * must be a primitive type or implement a 'toString' method.
 * @param props.children - Sub-components of the List component to add additional functionality.
 *
 */
export const List = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
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

type NestedKeys<T> =
  Cleanup<T> extends infer U
    ? U extends object
      ?
          | ValueOf<{ [K in keyof U]-?: (x: PrefixKeys<NestedKeys<U[K]>, K>) => void }>
          | ((x: U) => void) extends (x: infer I) => void
        ? { [K in keyof I]: I[K] }
        : never
      : U
    : never;

type Cleanup<T> = 0 extends 1 & T
  ? unknown
  : T extends readonly any[]
    ? Exclude<keyof T, keyof any[]> extends never
      ? Record<`${number}`, T[number]>
      : Omit<T, keyof any[]>
    : T;

type PrefixKeys<V, K extends PropertyKey> = V extends object
  ? {
      [P in keyof V as `${Extract<K, string | number>}.${Extract<P, string | number>}`]: V[P];
    }
  : { [P in K]: V };

type ValueOf<T> = T[keyof T];

type Test = NestedKeys<Example>;

const test: Test = {
  a: "test",
  dogs: {
    0: { name: "test" },
  },
  tasks: {
    channels: {
      0: {
        ports: {
          0: {
            line: "test",
          },
        },
      },
    },
  },
};

type T = ValueOf<Test>;
const v: T["tasks.channels.0.ports.0.line"] = "test";
