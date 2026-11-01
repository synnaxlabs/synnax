// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/user/Info.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

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

/** The logged-in user's name and roles, for embedding in a dialog. */
export const Info = (): ReactElement => {
  const { data: name } = User.useResultName({});
  return (
    <Flex.Box y gap="tiny" className={CSS.B("user-info")}>
      <Flex.Box x align="center" gap="small">
        <Text.Text weight={500} color={11} overflow="fade">
          {name}
        </Text.Text>
        <Roles />
      </Flex.Box>
    </Flex.Box>
  );
};
