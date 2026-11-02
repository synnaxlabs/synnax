// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Credential, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";

export interface SectionProps {
  /** The user whose password to set. */
  subject: ontology.ID;
  /** Called with true after a change, and with false once the user types again. */
  onChanged: (changed: boolean) => void;
}

/** Sets a user's password. The caller's own password needs the current one. */
export const Section = ({ subject, onChanged }: SectionProps): ReactElement => {
  const client = Synnax.use();
  const own = client?.auth.subject?.key === subject.key;
  const [password, setPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const set = Credential.Password.useSet({
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
      <Text.Text level="h5">Password</Text.Text>
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
