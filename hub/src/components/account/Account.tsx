// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Icon, Menu, Text, User } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useClerk, useUser } from "@/portal/ui/clerk";

/**
 * Account is the header control: a sign-in link when signed out, an avatar menu when
 * signed in.
 */
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
  if (user === undefined) return null;
  if (user === null)
    return (
      <Button.Button className="account-button" variant="outlined" href="/sign-in">
        Sign in
      </Button.Button>
    );
  const name = user.fullName ?? user.primaryEmailAddress?.emailAddress ?? "";
  return (
    <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
      <Dialog.Trigger
        variant="text"
        hideCaret
        square
        aria-label="Account menu"
        className="account-button account-avatar"
        style={user.hasImage ? undefined : { background: User.avatar(name) }}
      >
        {user.hasImage ? (
          <img className="account-avatar__image" src={user.imageUrl} alt="" />
        ) : (
          name.slice(0, 1).toUpperCase()
        )}
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        rounded
        background={1}
        className="account-menu"
        style={{ padding: "1rem", minWidth: "24rem" }}
      >
        <Text.Text level="small" color={9} style={{ padding: "1rem 2rem" }}>
          {name}
        </Text.Text>
        <Menu.Menu
          level="small"
          onChange={{
            licenses: () => window.location.assign("/account"),
            settings: () => window.location.assign("/account/settings"),
            signOut,
          }}
        >
          <Menu.Item itemKey="licenses">
            <Icon.Access />
            Licenses
          </Menu.Item>
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
