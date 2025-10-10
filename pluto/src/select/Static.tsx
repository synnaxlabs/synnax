// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional, type record } from "@synnaxlabs/x";

import { Component } from "@/component";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { ListItem } from "@/select/ListItem";
import { Single, type SingleProps } from "@/select/Single";

export interface StaticEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface StaticProps<
  K extends record.Key,
  E extends StaticEntry<K> = StaticEntry<K>,
> extends Optional<
      Omit<SingleProps<K, E>, "data" | "getItem" | "subscribe">,
      "children"
    >,
    List.UseStaticDataArgs<K, E> {}

const listItem = Component.renderProp((p: List.ItemProps<record.Key>) => {
  const { itemKey } = p;
  const item = List.useItem<record.Key, StaticEntry<record.Key>>(itemKey);
  if (item == null) return null;
  const { name, icon } = item;
  return (
    <ListItem {...p}>
      {icon}
      {name}
    </ListItem>
  );
});

export const Static = <K extends record.Key, E extends record.KeyedNamed<K>>({
  data,
  filter,
  children = listItem,
  virtual = false,
  ...rest
}: StaticProps<K, E>) => {
  const { retrieve, ...listProps } = List.useStaticData<K, E>({ data, filter });
  const { search } = List.usePager({ retrieve });
  return (
    <Single<K, E> {...rest} {...listProps} onSearch={search} virtual={virtual}>
      {children}
    </Single>
  );
};
