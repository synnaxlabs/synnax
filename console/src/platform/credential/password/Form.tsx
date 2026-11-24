// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/password/Form.css";

import { type auth, type resource, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Credential, User } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

/** Receives the result of a change, or null once the user types again. */
export type MessageHandler = (message: string | null) => void;

interface CreateFormProps {
  subject: resource.ID;
  onCreate: () => void;
  onMessage: MessageHandler;
}

const CreateForm = ({
  subject,
  onCreate,
  onMessage,
}: CreateFormProps): ReactElement => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const create = Credential.Password.useCreate({
    afterSuccess: () => {
      onMessage("Successfully created password");
      onCreate();
    },
  });
  Modals.useBusy(create.variant === "loading");
  const handleUsername = (value: string) => {
    setUsername(value);
    onMessage(null);
  };
  const handlePassword = (value: string) => {
    setPassword(value);
    onMessage(null);
  };
  return (
    <>
      <Input.Text
        value={username}
        onChange={handleUsername}
        placeholder="Username"
        full="x"
      />
      <Flex.Box x wrap justify="end">
        <Input.Text
          value={password}
          onChange={handlePassword}
          placeholder="Password"
          type="password"
          className={CSS.BE("credential", "password")}
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

interface RenameFormProps {
  password: auth.password.Password;
  /** Whether the password is the logged-in user's. */
  own: boolean;
  onMessage: MessageHandler;
}

const RenameForm = ({ password, own, onMessage }: RenameFormProps): ReactElement => {
  const [saved, setSaved] = useState(password.username);
  const [username, setUsername] = useState(password.username);
  const setCoreUsername = Core.useSetUsername();
  const rename = Credential.Password.useRename({
    afterSuccess: () => {
      setSaved(username);
      if (own) setCoreUsername(username);
      onMessage("Successfully changed username");
    },
  });
  Modals.useBusy(rename.variant === "loading");
  const handleUsername = (value: string) => {
    setUsername(value);
    onMessage(null);
  };
  return (
    <Flex.Box x wrap justify="end">
      <Input.Text
        value={username}
        onChange={handleUsername}
        placeholder="Username"
        className={CSS.BE("credential", "password")}
      />
      <Button.Button
        className={CSS.BE("credential", "action")}
        variant="filled"
        status={status.keepVariants(rename.variant, "loading")}
        disabled={username === "" || username === saved}
        onClick={() => rename.update({ key: password.key, username })}
      >
        Set username
      </Button.Button>
    </Flex.Box>
  );
};

interface SetFormProps {
  subject: resource.ID;
  onMessage: MessageHandler;
}

/**
 * Sets the subject's password. The logged-in user must give the current password to
 * change their own.
 */
const SetForm = ({ subject, onMessage }: SetFormProps): ReactElement => {
  const { data: currentKey } = User.useResultKey({});
  const own = currentKey === subject.key;
  const [password, setPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const dispatch = Session.useDispatch();
  const core = Session.Core.useSelectSelected();
  const set = Credential.Password.useSet({
    afterSuccess: () => {
      // a client built later from the record, as after a Core switch, logs in with it
      if (own && core != null && core.password !== "")
        dispatch(Session.Core.set({ ...core, password }));
      setPassword("");
      setCurrentPassword("");
      onMessage("Successfully changed password");
    },
  });
  Modals.useBusy(set.variant === "loading");
  const handleCurrentPassword = (value: string) => {
    setCurrentPassword(value);
    onMessage(null);
  };
  const handlePassword = (value: string) => {
    setPassword(value);
    onMessage(null);
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
      <Flex.Box x wrap justify="end">
        <Input.Text
          value={password}
          onChange={handlePassword}
          placeholder="New password"
          type="password"
          className={CSS.BE("credential", "password")}
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

interface PasswordsProps {
  subject: resource.ID;
  onCreate: () => void;
  onMessage: MessageHandler;
}

const Passwords = ({
  subject,
  onCreate,
  onMessage,
}: PasswordsProps): ReactElement | null => {
  const passwords = Credential.Password.useResult({ subject });
  const { data: currentKey } = User.useResultKey({});
  const own = currentKey === subject.key;
  // A rename needs update on the user, even for the user's own password.
  const canRename = Access.useUpdateGranted(subject);
  if (passwords.variant === "error")
    return <Text.Text status="error">{passwords.status.message}</Text.Text>;
  if (passwords.data == null) return null;
  if (passwords.data.length === 0)
    return <CreateForm subject={subject} onCreate={onCreate} onMessage={onMessage} />;
  return (
    <>
      {canRename && (
        <RenameForm password={passwords.data[0]} own={own} onMessage={onMessage} />
      )}
      <SetForm subject={subject} onMessage={onMessage} />
    </>
  );
};

export interface FormProps {
  /** The user whose password to create or change. */
  subject: resource.ID;
  onMessage: MessageHandler;
}

/**
 * Creates a password for a subject that has none. Otherwise sets the password, and
 * changes the username when the viewer may update the subject.
 */
export const Form = ({ subject, onMessage }: FormProps): ReactElement => {
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
      <Passwords
        key={generation}
        subject={subject}
        onCreate={handleCreate}
        onMessage={onMessage}
      />
    </Flex.Box>
  );
};
