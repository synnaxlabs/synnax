// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/user/Badge.css";

import {
  Access,
  Button,
  CSS as PCSS,
  Dialog,
  Divider,
  Flex,
  Icon,
  Tag,
  Text,
  User,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Clipboard } from "@/platform/clipboard";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

const Roles = (): ReactElement | null => {
  const { data: key } = User.useResultKey({});
  const { data: roles } = Access.Role.useResultForUser(
    key != null ? { user: key } : null,
  );
  if (roles == null || roles.length === 0) return null;
  return (
    <Flex.Box x wrap gap="small" className={CSS.BE("user-badge", "roles")}>
      {roles.map(({ key, name }) => (
        <Tag.Tag key={key} size="small" icon={<Icon.Role />}>
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

export const Badge = (): ReactElement | null => {
  const { data: remoteUsername } = User.useResultUsername({});
  const { data: firstName } = User.useResultFirstName({});
  const { data: lastName } = User.useResultLastName({});
  const core = Session.Core.useSelectSelected();
  const copy = Clipboard.useCopy();
  const handleLogout = Session.useLogout();
  const username = remoteUsername ?? core?.username ?? "";
  const fullName = [firstName, lastName].filter((part) => part).join(" ");
  const name = fullName !== "" ? fullName : username;
  return (
    <Dialog.Frame>
      <Dialog.Trigger aria-label="User menu" hideCaret textColor={10} gap="small">
        <Icon.User />
        {firstName != null && firstName !== "" ? firstName : username}
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        borderColor={7}
        className={CSS.BE("user-badge", "dialog")}
      >
        {core != null && (
          <>
            <Flex.Box y gap="tiny" className={CSS.BE("user-badge", "body")}>
              <Text.Text weight={500} color={10} overflow="ellipsis">
                {core.name}
              </Text.Text>
              <Flex.Box x align="center" gap="small" className={PCSS.M("reveals")}>
                <Text.Text level="small" color={9} overflow="ellipsis">
                  {core.host}:{core.port}
                </Text.Text>
                <Button.Button
                  variant="text"
                  size="tiny"
                  reveal
                  onClick={() => copy(`${core.host}:${core.port}`, "Core address")}
                >
                  <Icon.Copy />
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
            <Divider.Divider x />
          </>
        )}
        <Flex.Box y gap="tiny" className={CSS.BE("user-badge", "body")}>
          <Flex.Box x align="center" gap="small">
            <Text.Text weight={500} color={11} overflow="ellipsis">
              {name}
            </Text.Text>
            <Roles />
          </Flex.Box>
          {fullName !== "" && (
            <Text.Text level="small" color={9} overflow="ellipsis">
              {username}
            </Text.Text>
          )}
        </Flex.Box>
        <Flex.Box x className={CSS.BE("user-badge", "actions")}>
          <Button.Button
            onClick={handleLogout}
            variant="filled"
            status="error"
            size="small"
            justify="center"
            grow
          >
            <Icon.Logout />
            Log out
          </Button.Button>
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
