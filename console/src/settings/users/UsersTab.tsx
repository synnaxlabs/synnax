// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  Icon,
  List as PList,
  Select,
  Text,
  User,
} from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Layout } from "@/layout";
import { Item } from "@/user/Item";
import { REGISTER_LAYOUT } from "@/user/Register";

export const UsersTab = (): ReactElement => {
  const { data, getItem, subscribe } = User.useList({});
  const [selected, setSelected] = useState<string[]>([]);
  const placeLayout = Layout.usePlacer();

  return (
    <Flex.Box y gap="large" grow>
      <Flex.Box x justify="between" align="center">
        <Text.Text level="h3">Users</Text.Text>
        <Button.Button onClick={() => placeLayout(REGISTER_LAYOUT)}>
          <Icon.Add /> Add User
        </Button.Button>
      </Flex.Box>

      <Select.Frame<string, User.User>
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
      >
        <PList.Items<string> grow>
          {({ key, ...rest }) => <Item key={key} itemKey={key} {...rest} />}
        </PList.Items>
      </Select.Frame>
    </Flex.Box>
  );
};
