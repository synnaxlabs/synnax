// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/user/Info.css";

import { Access, Flex, Icon, Tag, Text, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

const Roles = (): ReactElement | null => {
  const { data: key } = User.useResultKey({});
  const { data: roles } = Access.Role.useResultForUser(
    key != null ? { user: key } : null,
  );
  if (roles == null || roles.length === 0) return null;
  return (
    <Flex.Box x wrap gap="small" className={CSS.BE("user-info", "roles")}>
      {roles.map(({ key, name }) => (
        <Tag.Tag key={key} size="small" icon={<Icon.Role />}>
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

/** The signed-in user's first name, falling back to their username. */
export const useDisplayName = (): string => {
  const { data: remoteUsername } = User.useResultUsername({});
  const { data: firstName } = User.useResultFirstName({});
  const core = Session.Core.useSelectSelected();
  const username = remoteUsername ?? core?.username ?? "";
  return firstName != null && firstName !== "" ? firstName : username;
};

/** The signed-in user's name, roles, and username, for embedding in a dialog. */
export const Info = (): ReactElement => {
  const { data: remoteUsername } = User.useResultUsername({});
  const { data: firstName } = User.useResultFirstName({});
  const { data: lastName } = User.useResultLastName({});
  const core = Session.Core.useSelectSelected();
  const username = remoteUsername ?? core?.username ?? "";
  const fullName = [firstName, lastName].filter((part) => part).join(" ");
  const name = fullName !== "" ? fullName : username;
  return (
    <Flex.Box y gap="tiny" className={CSS.B("user-info")}>
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
  );
};
