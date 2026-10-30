// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, type ontology, status, TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Credential, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Token } from "@/platform/credential/apikey/Token";

const sessionsLabel = (sessions: number | null): string => {
  if (sessions == null) return "sessions unknown";
  return `${sessions} live ${sessions === 1 ? "session" : "sessions"}`;
};

interface RowProps {
  apiKey: auth.apiKey.APIKey;
  /** The key's live sessions, or null when they could not be retrieved. */
  sessions: number | null;
  onDelete?: (key: auth.apiKey.Key) => void;
}

const Row = ({ apiKey, sessions, onDelete }: RowProps): ReactElement => (
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

interface ListProps {
  subject: ontology.ID;
  onDelete?: (key: auth.apiKey.Key) => void;
}

const List = ({ subject, onDelete }: ListProps): ReactElement => {
  const keys = Credential.APIKey.useResult({ subject });
  const sessions = Credential.Session.useResult({ subject });
  const sessionCount = (key: auth.apiKey.Key): number | null =>
    sessions.variant === "success"
      ? sessions.data.filter((s) => s.credential === key).length
      : null;
  return (
    <>
      {keys.variant === "error" && (
        <Text.Text status="error">{keys.status.message}</Text.Text>
      )}
      {keys.data?.map((k) => (
        <Row
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

export interface SectionProps {
  /** The user or rack whose API keys to manage. */
  subject: ontology.ID;
}

/** Lists a subject's API keys and lets the caller create and delete them. */
export const Section = ({ subject }: SectionProps): ReactElement => {
  const client = Synnax.use();
  // The Core checks permissions only for another subject's keys.
  const own = client?.auth.subject?.key === subject.key;
  const canCreate = Access.useUpdateGranted(subject) || own;
  const canDelete = Access.useDeleteGranted(subject) || own;
  const query = { subject };
  const invalidateAPIKeys = Credential.APIKey.useInvalidate();
  const invalidateSessions = Credential.Session.useInvalidate();
  const [name, setName] = useState("");
  const [created, setCreated] = useState<Credential.APIKey.CreateResult | null>(null);
  // These reads have no live updates, so a remount is what fetches them again.
  const [generation, setGeneration] = useState(0);
  const refresh = () => {
    invalidateAPIKeys(query);
    invalidateSessions(query);
    setGeneration((g) => g + 1);
  };
  const handleError = Status.useErrorHandler();
  const mounted = useRef(true);
  const create = Credential.APIKey.useCreate({
    afterSuccess: ({ data }) => {
      // a closed dialog can never show the token, so the key is of no use
      if (!mounted.current) {
        handleError(async () => {
          await client?.auth.apiKeys.delete(data.key.key);
          // a reopened dialog may have listed the key before this delete finished
          invalidateAPIKeys(query);
        }, "Failed to delete an unused API key");
        return;
      }
      setCreated(data);
      setName("");
      refresh();
    },
  });
  const del = Credential.APIKey.useDelete({
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
      invalidateAPIKeys(query);
      invalidateSessions(query);
    };
  }, []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">API keys</Text.Text>
      <List
        key={generation}
        subject={subject}
        onDelete={canDelete ? (k) => del.update(k) : undefined}
      />
      {canCreate && (
        <Flex.Box x>
          <Input.Text value={name} onChange={setName} placeholder="Key name" full="x" />
          <Button.Button
            variant="filled"
            status={status.keepVariants(create.variant, "loading")}
            onClick={() => create.update({ subject, name })}
          >
            <Icon.Add />
            Create key
          </Button.Button>
        </Flex.Box>
      )}
      {created != null && <Token token={created.token} />}
    </Flex.Box>
  );
};
