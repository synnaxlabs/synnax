// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/useModal.css";

import { type auth, type resource, status, TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Credential, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Token } from "@/platform/credential/Token";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

export interface ModalParams {
  /** The user or rack whose credentials to manage. */
  subject: resource.ID;
  title?: string | string[];
}

const sessionsLabel = (sessions: number | null): string => {
  if (sessions == null) return "sessions unknown";
  return `${sessions} live ${sessions === 1 ? "session" : "sessions"}`;
};

interface KeyRowProps {
  apiKey: auth.APIKey;
  /** The key's live sessions, or null when they could not be retrieved. */
  sessions: number | null;
  onDelete?: (key: auth.APIKeyKey) => void;
}

const KeyRow = ({ apiKey, sessions, onDelete }: KeyRowProps): ReactElement => (
  <Flex.Box x align="center" justify="between">
    <Flex.Box y gap="tiny">
      <Text.Text weight={500}>
        {apiKey.name === "" ? "Unnamed key" : apiKey.name}
      </Text.Text>
      <Text.Text level="small" color={9}>
        {`Created ${new TimeStamp(apiKey.created).toString("dateTime", "local")} · ${sessionsLabel(sessions)}`}
      </Text.Text>
    </Flex.Box>
    {onDelete != null && (
      <Button.Button
        variant="text"
        tooltip="Delete key"
        onClick={() => onDelete(apiKey.key)}
      >
        <Icon.Delete />
      </Button.Button>
    )}
  </Flex.Box>
);

interface KeyListProps extends Pick<ModalParams, "subject"> {
  onDelete?: (key: auth.APIKeyKey) => void;
}

const KeyList = ({ subject, onDelete }: KeyListProps): ReactElement => {
  const keys = Credential.useResultKeys({ subject });
  const sessions = Credential.useResultSessions({ subject });
  const sessionCount = (key: auth.APIKeyKey): number | null =>
    sessions.variant === "success"
      ? sessions.data.filter((s) => s.credential === key).length
      : null;
  return (
    <>
      {keys.variant === "error" && (
        <Text.Text status="error">{keys.status.message}</Text.Text>
      )}
      {keys.data?.map((k) => (
        <KeyRow
          key={k.key}
          apiKey={k}
          sessions={sessionCount(k.key)}
          onDelete={onDelete}
        />
      ))}
      {keys.data?.length === 0 && (
        <Text.Text level="small" color={9}>
          No API keys
        </Text.Text>
      )}
    </>
  );
};

const Keys = ({ subject }: Pick<ModalParams, "subject">): ReactElement => {
  const client = Synnax.use();
  // The Core checks permissions only for another subject's keys.
  const own = client?.auth.subject?.key === subject.key;
  const canCreate = Access.useUpdateGranted(subject) || own;
  const canDelete = Access.useDeleteGranted(subject) || own;
  const query = { subject };
  const invalidateKeys = Credential.useInvalidateKeys();
  const invalidateSessions = Credential.useInvalidateSessions();
  const refreshKeys = Credential.useRefreshKeys();
  const refreshSessions = Credential.useRefreshSessions();
  const [name, setName] = useState("");
  const [created, setCreated] = useState<Credential.CreateKeyResult | null>(null);
  // These reads have no live updates, so a change made here fetches them again.
  const refresh = () => {
    refreshKeys(query);
    refreshSessions(query);
  };
  const handleError = Status.useErrorHandler();
  const mounted = useRef(true);
  const create = Credential.useCreateKey({
    afterSuccess: ({ data }) => {
      // a closed dialog can never show the token, so the key is of no use
      if (!mounted.current) {
        handleError(async () => {
          await client?.auth.deleteKeys(data.key.key);
          // a reopened dialog may have listed the key before this delete finished
          invalidateKeys(query);
        }, "Failed to delete an unused API key");
        return;
      }
      setCreated(data);
      setName("");
      refresh();
    },
  });
  const del = Credential.useDeleteKey({
    afterSuccess: ({ data }) => {
      setCreated((c) => (c?.key.key === data ? null : c));
      refresh();
    },
  });
  // Drop the answers on close so the next open shows sessions opened or ended since.
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      invalidateKeys(query);
      invalidateSessions(query);
    };
  }, []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">API keys</Text.Text>
      <Flex.Box y className={CSS.BE("credential", "keys")}>
        <KeyList
          subject={subject}
          onDelete={canDelete ? (k) => del.update(k) : undefined}
        />
      </Flex.Box>
      {canCreate && (
        <Flex.Box x>
          <Input.Text value={name} onChange={setName} placeholder="Key name" full="x" />
          <Button.Button
            className={CSS.BE("credential", "action")}
            variant="filled"
            status={status.keepVariants(create.variant, "loading")}
            onClick={() => create.update({ subject, name })}
          >
            Create key
          </Button.Button>
        </Flex.Box>
      )}
      {created != null && <Token token={created.token} />}
    </Flex.Box>
  );
};

interface PasswordProps extends Pick<ModalParams, "subject"> {
  onChanged: (changed: boolean) => void;
}

const Password = ({ subject, onChanged }: PasswordProps): ReactElement => {
  const client = Synnax.use();
  const own = client?.auth.subject?.key === subject.key;
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

const Credentials = ({
  subject,
  title,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const [passwordChanged, setPasswordChanged] = useState(false);
  return (
    <Modal.Frame className={CSS.B("credential")}>
      <Modal.Header icon={<Icon.Lock />}>{title ?? "Sign-in"}</Modal.Header>
      <Modal.Body>
        <Flex.Box y gap="large">
          <Keys subject={subject} />
          {subject.type === "user" && (
            <Password subject={subject} onChanged={setPasswordChanged} />
          )}
        </Flex.Box>
      </Modal.Body>
      {passwordChanged && (
        <Modal.Footer>
          <Status.Summary variant="success" message="Successfully changed password" />
        </Modal.Footer>
      )}
    </Modal.Frame>
  );
};

export const useModal = Modals.create<ModalParams>(Credentials);
