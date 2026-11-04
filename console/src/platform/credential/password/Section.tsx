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
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/platform/css";

/** Receives the result of a change, or null once the user types again. */
type MessageHandler = (message: string | null) => void;

interface FormProps {
  subject: ontology.ID;
  /** Called after a write that changes what the form loaded. */
  onChange: () => void;
  onMessage: MessageHandler;
}

const CreateForm = ({ subject, onChange, onMessage }: FormProps): ReactElement => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const create = Credential.Password.useCreate({
    afterSuccess: () => {
      onMessage("Successfully created password");
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
          className={CSS.BE("credential", "action")}
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

interface RenameFormProps extends Pick<FormProps, "subject" | "onMessage"> {
  username: string;
}

const RenameForm = ({
  subject,
  username: initial,
  onMessage,
}: RenameFormProps): ReactElement => {
  const [saved, setSaved] = useState(initial);
  const [username, setUsername] = useState(initial);
  const handleUsername = (value: string) => {
    setUsername(value);
    onMessage(null);
  };
  const rename = Credential.Password.useRename({
    afterSuccess: () => {
      setSaved(username);
      onMessage("Successfully changed username");
    },
  });
  return (
    <Flex.Box x>
      <Input.Text
        value={username}
        onChange={handleUsername}
        placeholder="Username"
        full="x"
      />
      <Button.Button
        className={CSS.BE("credential", "action")}
        variant="filled"
        status={status.keepVariants(rename.variant, "loading")}
        disabled={username === "" || username === saved}
        onClick={() => rename.update({ subject, username })}
      >
        Set username
      </Button.Button>
    </Flex.Box>
  );
};

type SetFormProps = Pick<FormProps, "subject" | "onMessage">;

const SetForm = ({ subject, onMessage }: SetFormProps): ReactElement => {
  const [password, setPassword] = useState("");
  const set = Credential.Password.useSet({
    afterSuccess: () => {
      setPassword("");
      onMessage("Successfully changed password");
    },
  });
  const handlePassword = (value: string) => {
    setPassword(value);
    onMessage(null);
  };
  return (
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
        onClick={() => set.update({ subject, password })}
      >
        Set password
      </Button.Button>
    </Flex.Box>
  );
};

const Form = ({ subject, onChange, onMessage }: FormProps): ReactElement | null => {
  const passwords = Credential.Password.useResult({ subject });
  if (passwords.variant === "error")
    return <Text.Text status="error">{passwords.status.message}</Text.Text>;
  if (passwords.data == null) return null;
  if (passwords.data.length === 0)
    return <CreateForm subject={subject} onChange={onChange} onMessage={onMessage} />;
  return (
    <>
      <RenameForm
        subject={subject}
        username={passwords.data[0].username}
        onMessage={onMessage}
      />
      <SetForm subject={subject} onMessage={onMessage} />
    </>
  );
};

export interface SectionProps {
  /** The user whose password to create or set. */
  subject: ontology.ID;
  onMessage: MessageHandler;
}

/** Sets a user's username and password, or creates a password when they have none. */
export const Section = ({ subject, onMessage }: SectionProps): ReactElement => {
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
      <Form
        key={generation}
        subject={subject}
        onChange={handleChange}
        onMessage={onMessage}
      />
    </Flex.Box>
  );
};
