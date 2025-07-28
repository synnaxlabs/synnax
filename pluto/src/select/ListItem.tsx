// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { List } from "@/list";
import { useItemState } from "@/select/Frame";

export interface ListItemProps<K extends record.Key = record.Key>
  extends List.ItemProps<K> {}

export const ListItem = <K extends record.Key = record.Key>(
  props: ListItemProps<K>,
) => {
  const { itemKey } = props;
  const selectProps = useItemState(itemKey);
  return <List.Item {...selectProps} {...props} allowSelect />;
};
