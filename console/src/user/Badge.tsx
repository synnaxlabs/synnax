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
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

export const Badge = (): ReactElement | null => {
  const dispatch = useDispatch();
  const { data: u } = User.useRetrieve({});
  const handleLogout = useCallback(() => {
    dispatch(Cluster.setActive(null));
    dispatch(Workspace.setActive(null));
    dispatch(Layout.clearWorkspace());
  }, [dispatch]);
  if (u == null) return null;
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
            {u.firstName != "" ? `${u.firstName}` : u.username}
          </Dialog.Trigger>
        </Flex.Box>
      </Flex.Box>
      <Dialog.Dialog bordered borderColor={6} style={{ padding: "1rem", width: 200 }}>
        <Button.Button onClick={handleLogout} variant="text" full="x">
          <Icon.Logout />
          Log out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
