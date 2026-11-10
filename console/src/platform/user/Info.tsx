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

const Role = (): ReactElement | null => {
  const { data: key } = User.useResultKey({});
  const { data: role } = Access.Role.useResultForUser(
    key != null ? { user: key } : null,
  );
  if (role == null) return null;
  return (
    <Tag.Tag size="small" icon={<Icon.Role />} className={CSS.BE("user-info", "role")}>
      {role.name}
    </Tag.Tag>
  );
};

/** The logged-in user's name and role, for embedding in a dialog. */
export const Info = (): ReactElement => {
  const { data: name } = User.useResultName({});
  return (
    <Flex.Box y gap="tiny" className={CSS.B("user-info")}>
      <Flex.Box x align="center" gap="small">
        <Text.Text weight={500} color={11} overflow="fade">
          {name}
        </Text.Text>
        <Role />
      </Flex.Box>
    </Flex.Box>
  );
};
