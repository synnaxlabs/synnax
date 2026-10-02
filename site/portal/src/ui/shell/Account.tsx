// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Avatar } from "@synnaxlabs/lyra/avatar";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { useClerk } from "@/ui/clerk";

export interface AccountProps {
  name: string;
  /** image is the URL of the user's picture. Initials show without one. */
  image?: string;
}

/** Account is the avatar menu in the portal bar: the user's settings and log out. */
export const Account = ({ name, image }: AccountProps): ReactElement => {
  const clerk = useClerk();
  const logout = (): void => {
    if (clerk == null) return;
    void clerk.signOut(() => {
      window.location.assign("/");
      return Promise.resolve();
    });
  };
  return (
    <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
      <Dialog.Trigger
        variant="text"
        hideCaret
        aria-label="Account menu"
        className="portal-account__trigger"
      >
        <Avatar.Avatar name={name} image={image} />
      </Dialog.Trigger>
      <Dialog.Dialog bordered rounded background={1} className="portal-menu">
        <Text.Text level="small" color={9} className="portal-menu__name">
          {name}
        </Text.Text>
        <Menu.Menu
          level="small"
          onChange={{
            settings: () => window.location.assign("/settings"),
            logout,
          }}
        >
          <Menu.Item itemKey="settings">
            <Icon.Settings />
            Settings
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item itemKey="logout">
            <Icon.Logout />
            Log out
          </Menu.Item>
        </Menu.Menu>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
