// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/Form.css";

import { type auth, type resource, status, TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Credential, Synnax, User } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Token } from "@/platform/credential/Token";
import { CSS } from "@/platform/css";

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

interface KeyListProps {
  subject: resource.ID;
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

export interface KeysProps {
  /** The user or rack whose API keys to manage. */
  subject: resource.ID;
}

/**
 * Lists the subject's API keys, with their live sessions, and creates and deletes keys
 * as the user's access allows. A new key's token shows once, until the next create.
 */
export const Keys = ({ subject }: KeysProps): ReactElement => {
  const client = Synnax.use();
  const { data: currentKey } = User.useResultKey({});
  // The Core checks permissions only for another subject's keys.
  const own = currentKey === subject.key;
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
      // an unmounted list can never show the token, so the key is of no use
      if (!mounted.current) {
        handleError(async () => {
          await client?.auth.deleteKeys(data.key.key);
          // a remounted list may have listed the key before this delete finished
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
  // Drop the answers on unmount so the next mount shows sessions opened or ended since.
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
