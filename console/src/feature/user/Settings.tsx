// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/user/Settings.css";

import { user } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Synnax, User } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Core } from "@/platform/core";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Page } from "@/platform/page";
import { Settings as PlatformSettings } from "@/platform/settings";
import { Session } from "@/session";

// Any user may change their own password and link or unlink their own accounts.
const SignIn = ({ userKey }: { userKey: user.Key }): ReactElement | null => {
  const client = Synnax.use();
  const { password, oidc } = Core.useAcceptedMethods(client?.params);
  const [message, setMessage] = useState<string | null>(null);
  if (!password && !oidc) return null;
  const subject = user.resourceID(userKey);
  return (
    <Page.Card title="Sign-in">
      <PlatformSettings.Body>
        <Flex.Box y gap="large">
          {password && (
            <Flex.Box y>
              <Text.Text level="h5">Password</Text.Text>
              <Credential.Password.Form subject={subject} onMessage={setMessage} />
            </Flex.Box>
          )}
          {oidc && <Credential.OIDC.Section subject={subject} onMessage={setMessage} />}
        </Flex.Box>
        {message != null && <Status.Summary variant="success" message={message} />}
      </PlatformSettings.Body>
    </Page.Card>
  );
};

interface NameProps {
  current: user.User;
}

// Only a user who can update users may rename themselves.
const Name = ({ current }: NameProps): ReactElement => {
  const canRename = Access.useUpdateGranted(user.resourceID(current.key));
  const { update: rename } = User.useRename();
  const handleRename = (name: string): void => {
    if (name !== current.name) rename({ key: current.key, name });
  };
  return (
    <PlatformSettings.Entry label="Name" description="Other users see this name.">
      {canRename ? (
        <Input.Text
          aria-label="Name"
          value={current.name}
          onChange={handleRename}
          onlyChangeOnBlur
          resetOnBlurIfEmpty
          className={CSS.BE("user-settings", "name")}
        />
      ) : (
        <Text.Text color={10}>{current.name}</Text.Text>
      )}
    </PlatformSettings.Entry>
  );
};

/** The logged-in user's name, username, password, and provider accounts. */
export const ProfileSettings = (): ReactElement | null => {
  const { data: current } = User.useResult({});
  const core = Session.Core.useSelectSelected();
  if (current == null) return null;
  return (
    <>
      <Page.Card>
        <Name current={current} />
        {core != null && core.username !== "" && (
          <PlatformSettings.Entry label="Username" description="You log in with it.">
            <Text.Text color={10}>{core.username}</Text.Text>
          </PlatformSettings.Entry>
        )}
      </Page.Card>
      <SignIn userKey={current.key} />
    </>
  );
};

/** The logged-in user's API keys. */
export const KeySettings = (): ReactElement | null => {
  const { data: key } = User.useResultKey({});
  if (key == null) return null;
  return <Credential.APIKey.Table subject={user.resourceID(key)} />;
};
