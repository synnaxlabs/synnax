// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/user/Badge.css";

import { Access, Button, Dialog, Divider, Flex, Icon, User } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { Settings } from "@/settings";
import { Workspace } from "@/workspace";

export const Badge = (): ReactElement | null => {
  const dispatch = useDispatch();
  const { data: u } = User.useRetrieve({}, { addStatusOnFailure: false });
  const cluster = Cluster.useSelect();
  const { data: u } = User.useRetrieve({});
  const isAdmin = Access.useIsAdmin();
  const placeLayout = Layout.usePlacer();

  const handleLogout = useCallback(() => {
    dispatch(Cluster.setActive(null));
    dispatch(Workspace.setActive(null));
    dispatch(Layout.clearWorkspace());
  }, [dispatch]);
  const username = u?.username ?? cluster?.username ?? "";
  const displayName =
    u?.firstName != null && u?.firstName != "" ? u.firstName : username;
  return (
    <Dialog.Frame>
      <Dialog.Trigger contrast={2} hideCaret textColor={10} gap="small" weight={400}>
        <Icon.User />
        {displayName}
      </Dialog.Trigger>
      <Dialog.Dialog bordered borderColor={6} style={{ padding: "1rem", width: 200 }}>
        {isAdmin && (
          <>
            <Button.Button
              onClick={() => placeLayout(Settings.CLUSTER_SETTINGS_LAYOUT)}
              variant="text"
              full="x"
            >
              <Icon.Settings />
              Cluster Settings
            </Button.Button>
            <Divider.Divider />
          </>
        )}
        <Button.Button onClick={handleLogout} variant="text" full="x">
          <Icon.Logout />
          Log out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
