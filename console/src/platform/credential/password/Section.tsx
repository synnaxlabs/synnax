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

interface FormProps {
  subject: ontology.ID;
  /** Called after a write that changes what the form loaded. */
  onChange: () => void;
}

const CreateForm = ({ subject, onChange }: FormProps): ReactElement => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const addStatus = Status.useAdder();
  const create = Credential.Password.useCreate({
    afterSuccess: () => {
      addStatus({ variant: "success", message: "Successfully created password" });
      onChange();
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

interface RenameFormProps extends FormProps {
  username: string;
}

const RenameForm = ({
  subject,
  username: initial,
  onChange,
}: RenameFormProps): ReactElement => {
  const [username, setUsername] = useState(initial);
  const addStatus = Status.useAdder();
  const rename = Credential.Password.useRename({
    afterSuccess: () => {
      addStatus({ variant: "success", message: "Successfully changed username" });
      onChange();
    },
  });
  return (
    <Flex.Box x>
      <Input.Text
        value={username}
        onChange={setUsername}
        placeholder="Username"
        full="x"
      />
      <Button.Button
        variant="filled"
        status={status.keepVariants(rename.variant, "loading")}
        disabled={username === "" || username === initial}
        onClick={() => rename.update({ subject, username })}
      >
        Set username
      </Button.Button>
    </Flex.Box>
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

const Form = ({ subject, onChange }: FormProps): ReactElement | null => {
  const passwords = Credential.Password.useResult({ subject });
  if (passwords.variant === "error")
    return <Text.Text status="error">{passwords.status.message}</Text.Text>;
  if (passwords.data == null) return null;
  if (passwords.data.length === 0)
    return <CreateForm subject={subject} onChange={onChange} />;
  return (
    <>
      <RenameForm
        subject={subject}
        username={passwords.data[0].username}
        onChange={onChange}
      />
      <SetForm subject={subject} />
    </>
  );
};

export interface SectionProps {
  /** The user whose password to create or set. */
  subject: ontology.ID;
}

/**
 * Sets a user's username and password, or creates a password for a user who has none.
 * The caller's own password needs the current one to set it.
 */
export const Section = ({ subject }: SectionProps): ReactElement => {
  const query = { subject };
  const invalidate = Credential.Password.useInvalidate();
  // This read has no live updates, so a remount is what fetches it again.
  const [generation, setGeneration] = useState(0);
  const handleChange = () => {
    invalidate(query);
    setGeneration((g) => g + 1);
  };
  useEffect(() => () => invalidate(query), []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">Password</Text.Text>
      <Form key={generation} subject={subject} onChange={handleChange} />
    </Flex.Box>
  );
};
