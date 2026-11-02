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
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Credential, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

interface CreateFormProps {
  subject: ontology.ID;
  onCreate: () => void;
}

const CreateForm = ({ subject, onCreate }: CreateFormProps): ReactElement => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const addStatus = Status.useAdder();
  const create = Credential.Password.useCreate({
    afterSuccess: () => {
      addStatus({ variant: "success", message: "Successfully created password" });
      onCreate();
    },
  });
  return (
    <>
      <Input.Text
        value={username}
        onChange={setUsername}
        placeholder="Username"
        full="x"
      />
      <Flex.Box x>
        <Input.Text
          value={password}
          onChange={setPassword}
          placeholder="Password"
          type="password"
          full="x"
        />
        <Button.Button
          variant="filled"
          status={status.keepVariants(create.variant, "loading")}
          disabled={username === "" || password === ""}
          onClick={() => create.update({ subject, username, password })}
        >
          Create password
        </Button.Button>
      </Flex.Box>
    </>
  );
};

interface SetFormProps {
  subject: ontology.ID;
}

const SetForm = ({ subject }: SetFormProps): ReactElement => {
  const client = Synnax.use();
  const own = client?.auth.subject?.key === subject.key;
  const [password, setPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const addStatus = Status.useAdder();
  const set = Credential.Password.useSet({
    afterSuccess: () => {
      setPassword("");
      setCurrentPassword("");
      addStatus({ variant: "success", message: "Successfully changed password" });
    },
  });
  const handleSet = () =>
    set.update(own ? { password, currentPassword } : { subject, password });
  return (
    <>
      {own && (
        <Input.Text
          value={currentPassword}
          onChange={setCurrentPassword}
          placeholder="Current password"
          type="password"
          full="x"
        />
      )}
      <Flex.Box x>
        <Input.Text
          value={password}
          onChange={setPassword}
          placeholder="New password"
          type="password"
          full="x"
        />
        <Button.Button
          variant="filled"
          status={status.keepVariants(set.variant, "loading")}
          disabled={password === ""}
          onClick={handleSet}
        >
          Set password
        </Button.Button>
      </Flex.Box>
    </>
  );
};

const Form = ({ subject, onCreate }: CreateFormProps): ReactElement | null => {
  const passwords = Credential.Password.useResult({ subject });
  if (passwords.variant === "error")
    return <Text.Text status="error">{passwords.status.message}</Text.Text>;
  if (passwords.data == null) return null;
  if (passwords.data.length === 0)
    return <CreateForm subject={subject} onCreate={onCreate} />;
  return <SetForm subject={subject} />;
};

export interface SectionProps {
  /** The user whose password to create or set. */
  subject: ontology.ID;
}

/**
 * Sets a user's password, or creates one for a user who has none. The caller's own
 * password needs the current one to set it.
 */
export const Section = ({ subject }: SectionProps): ReactElement => {
  const query = { subject };
  const invalidate = Credential.Password.useInvalidate();
  // This read has no live updates, so a remount is what fetches it again.
  const [generation, setGeneration] = useState(0);
  const handleCreate = () => {
    invalidate(query);
    setGeneration((g) => g + 1);
  };
  useEffect(() => () => invalidate(query), []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">Password</Text.Text>
      <Form key={generation} subject={subject} onCreate={handleCreate} />
    </Flex.Box>
  );
};
