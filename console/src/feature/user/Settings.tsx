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
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, User } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Settings as PlatformSettings } from "@/platform/settings";
import { Session } from "@/session";

const Password = ({ userKey }: { userKey: user.Key }): ReactElement => {
  const [changed, setChanged] = useState(false);
  return (
    <PlatformSettings.Card title="Password">
      <PlatformSettings.Body>
        <Credential.Password
          subject={user.resourceID(userKey)}
          onChanged={setChanged}
        />
        {changed && <Status.Summary variant="success" message="Changed password" />}
      </PlatformSettings.Body>
    </PlatformSettings.Card>
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

/** The logged-in user's name, username, and password. */
export const ProfileSettings = (): ReactElement | null => {
  const { data: current } = User.useResult({});
  const core = Session.Core.useSelectSelected();
  if (current == null) return null;
  return (
    <>
      <PlatformSettings.Card>
        <Name current={current} />
        {core != null && (
          <PlatformSettings.Entry label="Username" description="You log in with it.">
            <Text.Text color={10}>{core.username}</Text.Text>
          </PlatformSettings.Entry>
        )}
      </PlatformSettings.Card>
      <Password userKey={current.key} />
    </>
  );
};

/** The logged-in user's API keys. */
export const KeySettings = (): ReactElement | null => {
  const { data: key } = User.useResultKey({});
  if (key == null) return null;
  return (
    <PlatformSettings.Card>
      <PlatformSettings.Body>
        <Credential.Keys subject={user.resourceID(key)} />
      </PlatformSettings.Body>
    </PlatformSettings.Card>
  );
};
