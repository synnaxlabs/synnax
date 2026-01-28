// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { type Button } from "@/button";
import { List } from "@/list";
import { useItemState } from "@/select/Frame";

export type ListItemProps<
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
> = List.ItemProps<K, E>;

export const ListItem = <
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
>(
  props: ListItemProps<K, E>,
) => {
  const { itemKey } = props;
  const selectProps = useItemState(itemKey);
  return <List.Item<K, E> {...selectProps} {...props} />;
};
