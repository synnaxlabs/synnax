// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback } from "react";

import { useClerk, useUser } from "@/ui/clerk";
import { Mark } from "@/ui/shell/Mark";

/** Account is the avatar menu in the portal bar: the user's settings and sign out. */
export const Account = (): ReactElement | null => {
  const clerk = useClerk();
  const user = useUser();
  const signOut = useCallback(() => {
    if (clerk == null) return;
    void clerk.signOut(() => {
      window.location.assign("/");
      return Promise.resolve();
    });
  }, [clerk]);
  if (user == null) return null;
  const name = user.fullName ?? user.primaryEmailAddress?.emailAddress ?? "";
  return (
    <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
      <Dialog.Trigger
        variant="text"
        hideCaret
        aria-label="Account menu"
        className="portal-bar__mark-trigger"
      >
        <Mark name={name} image={user.hasImage ? user.imageUrl : undefined} />
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        rounded
        background={1}
        className="portal-menu"
        style={{ padding: "1rem", minWidth: "24rem" }}
      >
        <Text.Text level="small" color={9} style={{ padding: "1rem 2rem" }}>
          {name}
        </Text.Text>
        <Menu.Menu
          level="small"
          onChange={{
            settings: () => window.location.assign("/settings"),
            signOut,
          }}
        >
          <Menu.Item itemKey="settings">
            <Icon.Settings />
            Settings
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item itemKey="signOut">
            <Icon.Logout />
            Sign out
          </Menu.Item>
        </Menu.Menu>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
