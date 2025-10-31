// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/user/Badge.css";

import { Button, Dialog, Flex, Icon, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { ConnectionBadge } from "@/cluster/Badges";
import { setActive } from "@/cluster/slice";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

export const Badge = (): ReactElement => {
  const dispatch = useDispatch();
  const { data: u } = User.useRetrieve({});
  let text = u?.username;
  if (u?.firstName != "") text = `${u?.firstName}`;
  return (
    <Dialog.Frame>
      <Flex.Box x>
        <Flex.Box x pack>
          <User.Avatar username={u?.username ?? ""} square size="large" />
          <Dialog.Trigger
            contrast={2}
            hideCaret
            textColor={10}
            gap="small"
            weight={400}
          >
            {text}
          </Dialog.Trigger>
        </Flex.Box>
        <ConnectionBadge />
      </Flex.Box>
      <Dialog.Dialog bordered borderColor={6} style={{ padding: "1rem", width: 200 }}>
        <Button.Button
          onClick={() => {
            dispatch(Workspace.setActive(null));
            dispatch(setActive(null));
            dispatch(Layout.clearWorkspace());
          }}
          variant="text"
          full="x"
        >
          <Icon.Logout />
          Log out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
