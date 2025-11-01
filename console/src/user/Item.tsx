// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, List, Select, Tag, Text, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export const Item = ({
  itemKey,
  ...props
}: List.ItemProps<string>): ReactElement | null => {
  const user = List.useItem<string, User.User>(itemKey);
  const { selected, onSelect, ...selectProps } = Select.useItemState(itemKey);

  if (user == null) return null;

  return (
    <List.Item
      {...props}
      {...selectProps}
      selected={selected}
      rounded={!selected}
      onSelect={onSelect}
      justify="between"
      align="center"
    >
      <Flex.Box x gap="medium" align="center">
        <User.Avatar username={user.username} size="medium" />
        <Flex.Box y gap="tiny">
          <Text.Text level="p" weight={500}>
            {user.firstName} {user.lastName}
          </Text.Text>
          <Text.Text level="small" color="gray">
            @{user.username}
          </Text.Text>
          {/* TODO: Add role badges once useUserRoles hook is implemented */}
        </Flex.Box>
      </Flex.Box>
    </List.Item>
  );
};
