// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Access, Icon, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { EmptyAction, Toolbar } from "@/component";
import { type Service } from "@/service";
import { useRegisterModal } from "@/service/user/useRegisterModal";
import { Ontology } from "@/ontology";

const Content = (): ReactElement => {
  const { data: groupID } = User.useRetrieveGroupID({});
  const openRegister = useRegisterModal();
  const hasCreatePermission = Access.useCreateGranted(user.TYPE_ONTOLOGY_ID);
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.User />}>Users</Toolbar.Title>
        {hasCreatePermission && (
          <Toolbar.Actions>
            <Toolbar.Action onClick={() => openRegister()} tooltip="Create user">
              <Icon.Add />
            </Toolbar.Action>
          </Toolbar.Actions>
        )}
      </Toolbar.Header>
      <Ontology.Tree root={groupID} emptyContent={<EmptyContent />} />
    </Toolbar.Content>
  );
};

const EmptyContent = (): ReactElement => {
  const openRegister = useRegisterModal();
  const hasCreatePermission = Access.useCreateGranted(user.TYPE_ONTOLOGY_ID);
  return (
    <EmptyAction
      message="No users."
      action={hasCreatePermission ? "Create a user" : undefined}
      onClick={() => openRegister()}
    />
  );
};

export const TOOLBAR: Service.Nav.Item = {
  key: "user",
  icon: <Icon.User />,
  content: <Content />,
  tooltip: "Users",
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  trigger: ["U"],
  useVisible: () => Access.useUpdateGranted(user.TYPE_ONTOLOGY_ID),
};
