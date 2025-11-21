// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { POLICY_CREATE_LAYOUT } from "@/access/policy/Create";
import { ROLE_CREATE_LAYOUT } from "@/access/role/Create";
import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { REGISTER_LAYOUT } from "@/user/Register";

const Content = (): ReactElement => {
  const { data: groupID } = User.useRetrieveGroupID({});
  const placeLayout = Layout.usePlacer();
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.User />}>Users</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action
            onClick={() => placeLayout(REGISTER_LAYOUT)}
            tooltip="Create User"
          >
            <Icon.User />
            <Icon.Add />
          </Toolbar.Action>
          <Toolbar.Action
            onClick={() => placeLayout(ROLE_CREATE_LAYOUT)}
            tooltip="Create Role"
          >
            <Icon.Role />
            <Icon.Add />
          </Toolbar.Action>
          <Toolbar.Action
            onClick={() => placeLayout(POLICY_CREATE_LAYOUT)}
            tooltip="Create Policy"
          >
            <Icon.Policy />
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={groupID} />
    </Toolbar.Content>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "user",
  icon: <Icon.User />,
  content: <Content />,
  tooltip: "Users",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
  trigger: ["U"],
};
