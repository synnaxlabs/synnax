// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";
import { Component, List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface ItemProps extends List.ItemProps<workspace.Key> {}

export const Item = (props: ItemProps): ReactElement | null => {
  const { itemKey } = props;
  const workspace = List.useItem<workspace.Key, workspace.Workspace>(itemKey);
  console.log(workspace);
  if (workspace == null) return null;
  return (
    <Select.ListItem {...props}>
      <Text.Text level="p">{workspace.name}</Text.Text>
    </Select.ListItem>
  );
};

export const item = Component.renderProp(Item);
