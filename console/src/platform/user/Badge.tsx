// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/user/Badge.css";

import { Button, Dialog, Icon, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

export const Badge = (): ReactElement | null => {
  const { data: u } = User.useRetrieve({}, { addStatusOnFailure: false });
  const cluster = Session.Cluster.useSelectState();
  const handleLogout = Session.useLogout();
  const username = u?.username ?? cluster?.username ?? "";
  const displayName =
    u?.firstName != null && u?.firstName != "" ? u.firstName : username;
  return (
    <Dialog.Frame>
      <Dialog.Trigger hideCaret textColor={10} gap="small" weight={400}>
        <Icon.User />
        {displayName}
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        borderColor={7}
        className={CSS.BE("user-badge", "dialog")}
      >
        <Button.Button onClick={handleLogout} variant="text" full="x">
          <Icon.Logout />
          Log out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
