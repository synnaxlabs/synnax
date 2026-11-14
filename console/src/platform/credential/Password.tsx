// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/Form.css";

import { type resource, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Credential, User } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";

export interface PasswordProps {
  /** The user whose password to set. */
  subject: resource.ID;
  /** Called with true after a change succeeds, and with false on the next edit. */
  onChanged: (changed: boolean) => void;
}

/**
 * Sets the subject's password. The logged-in user must give the current password to
 * change their own.
 */
export const Password = ({ subject, onChanged }: PasswordProps): ReactElement => {
  const { data: currentKey } = User.useResultKey({});
  const own = currentKey === subject.key;
  const [password, setPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const set = Credential.useSetPassword({
    afterSuccess: () => {
      setPassword("");
      setCurrentPassword("");
      onChanged(true);
    },
  });
  const handleCurrentPassword = (value: string) => {
    setCurrentPassword(value);
    onChanged(false);
  };
  const handlePassword = (value: string) => {
    setPassword(value);
    onChanged(false);
  };
  const handleSet = () =>
    set.update(own ? { password, currentPassword } : { subject, password });
  return (
    <Flex.Box y>
      {own && (
        <Input.Text
          value={currentPassword}
          onChange={handleCurrentPassword}
          placeholder="Current password"
          type="password"
          full="x"
        />
      )}
      <Flex.Box x>
        <Input.Text
          value={password}
          onChange={handlePassword}
          placeholder="New password"
          type="password"
          full="x"
        />
        <Button.Button
          className={CSS.BE("credential", "action")}
          variant="filled"
          status={status.keepVariants(set.variant, "loading")}
          disabled={password === ""}
          onClick={handleSet}
        >
          Set password
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
