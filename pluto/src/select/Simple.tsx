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
import { Flux } from "@/flux";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Single, type SingleProps } from "@/select/Single";
import { Text } from "@/text";

export interface SimplyEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface SimpleProps<
  K extends record.Key,
  E extends SimplyEntry<K> = SimplyEntry<K>,
> extends Optional<
      Omit<SingleProps<K, E>, "data" | "getItem" | "subscribe">,
      "children"
    >,
    List.UseStaticDataArgs<K, E> {}

const listItem = Component.renderProp((p: List.ItemProps<record.Key>) => {
  const { itemKey } = p;
  const item = List.useItem<record.Key, SimplyEntry<record.Key>>(itemKey);
  if (item == null) return null;
  const { name, icon } = item;
  return (
    <Select.ListItem {...p}>
      <Text.WithIcon level="p" startIcon={icon}>
        {name}
      </Text.WithIcon>
    </Select.ListItem>
  );
});

export const Simple = <K extends record.Key, E extends record.KeyedNamed<K>>({
  data,
  filter,
  children = listItem,
  ...rest
}: SimpleProps<K, E>) => {
  const { retrieve, ...listProps } = List.useStaticData<K, E>({ data, filter });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Single<K, E>
      {...rest}
      {...listProps}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
    >
      {children}
    </Single>
  );
};
