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
import { Text } from "@synnaxlabs/lyra/text";
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { Common } from "@/platform/credential/common";

interface LiveProps {
  subject: ontology.ID;
  createStatus: status.Variant;
  changing: boolean;
  onCreate: () => void;
  onRevoke: (key: string) => void;
}

const Live = ({
  subject,
  createStatus,
  changing,
  onCreate,
  onRevoke,
}: LiveProps): ReactElement => {
  const invites = Credential.OIDC.Invite.useResult({ subject });
  const live = invites.data?.at(0);
  return (
    <>
      {invites.variant === "error" && (
        <Text.Text status="error">{invites.status.message}</Text.Text>
      )}
      {live != null && (
        <Flex.Box x align="center" justify="between">
          <Text.Text level="small" color={9}>
            {`Invite expires ${live.expiry.toString("dateTime", "local")}`}
          </Text.Text>
          <Button.Button
            variant="text"
            disabled={changing}
            onClick={() => onRevoke(live.key)}
          >
            Revoke
          </Button.Button>
        </Flex.Box>
      )}
      {invites.variant !== "loading" && (
        <Button.Button
          variant="outlined"
          full="x"
          justify="center"
          status={status.keepVariants(createStatus, "loading")}
          disabled={changing}
          onClick={onCreate}
        >
          {live == null ? "Create invite" : "Replace invite"}
        </Button.Button>
      )}
    </>
  );
};

export interface InviteProps {
  /** The user to invite. */
  subject: ontology.ID;
  /** Receives the result of a change, or null when a new one starts. */
  onMessage: (message: string | null) => void;
}

/**
 * Creates and revokes the one-time code that links a provider account to a user. The
 * Core replaces a user's live invite with each new one.
 */
export const Invite = ({ subject, onMessage }: InviteProps): ReactElement => {
  const query = { subject };
  const invalidate = Credential.OIDC.Invite.useInvalidate();
  const [token, setToken] = useState<string | null>(null);
  // This read has no live updates, so a remount is what fetches it again.
  const [generation, setGeneration] = useState(0);
  const handleChange = (message: string) => {
    invalidate(query);
    setGeneration((g) => g + 1);
    onMessage(message);
  };
  const create = Credential.OIDC.Invite.useCreate({
    afterSuccess: ({ data }) => {
      setToken(data.token);
      handleChange("Successfully created invite");
    },
  });
  const revoke = Credential.OIDC.Invite.useRevoke({
    afterSuccess: () => {
      setToken(null);
      handleChange("Successfully revoked invite");
    },
  });
  const handleCreate = () => {
    onMessage(null);
    create.update(subject);
  };
  const handleRevoke = (key: string) => {
    onMessage(null);
    revoke.update(key);
  };
  useEffect(() => () => invalidate(query), []);
  return (
    <Flex.Box y gap="small">
      <Live
        key={generation}
        subject={subject}
        createStatus={create.variant}
        changing={create.variant === "loading" || revoke.variant === "loading"}
        onCreate={handleCreate}
        onRevoke={handleRevoke}
      />
      {token != null && <Common.Token token={token} label="invite code" />}
    </Flex.Box>
  );
};
