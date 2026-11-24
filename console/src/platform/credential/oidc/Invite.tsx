// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { Code } from "@/platform/code";
import { Modals } from "@/platform/modals";

/** What the caller may do with the user's invite. */
export interface Grants {
  /** Creating an invite for a user with none. */
  create: boolean;
  /** Replacing a live invite with a new one. */
  replace: boolean;
  /** Revoking a live invite. */
  revoke: boolean;
}

interface LiveProps {
  subject: resource.ID;
  grants: Grants;
  createStatus: status.Variant;
  changing: boolean;
  onCreate: () => void;
  onRevoke: (key: string) => void;
}

const Live = ({
  subject,
  grants,
  createStatus,
  changing,
  onCreate,
  onRevoke,
}: LiveProps): ReactElement => {
  const invites = Credential.OIDC.Invite.useResult({ subject });
  const live = invites.data?.at(0);
  // An expired invite stays stored and hidden, and a new one writes over it, so the
  // Core may ask for either action when no live invite shows.
  const canCreate = live == null ? grants.create && grants.replace : grants.replace;
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
          {grants.revoke && (
            <Button.Button
              variant="text"
              disabled={changing}
              onClick={() => onRevoke(live.key)}
            >
              Revoke
            </Button.Button>
          )}
        </Flex.Box>
      )}
      {invites.variant !== "loading" && canCreate && (
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
  subject: resource.ID;
  grants: Grants;
  /** Receives the result of a change, or null when a new one starts. */
  onMessage: (message: string | null) => void;
}

/**
 * Creates and revokes the one-time code that links a provider account to a user. The
 * Core replaces a user's live invite with each new one.
 */
export const Invite = ({ subject, grants, onMessage }: InviteProps): ReactElement => {
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
  // Closing during a create would lose the only copy of the code.
  Modals.useBusy(create.variant === "loading");
  const handleCreate = () => {
    onMessage(null);
    create.update(subject.key);
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
        grants={grants}
        createStatus={create.variant}
        changing={create.variant === "loading" || revoke.variant === "loading"}
        onCreate={handleCreate}
        onRevoke={handleRevoke}
      />
      {token != null && (
        <Flex.Box y gap="tiny">
          <Text.Text level="small" status="warning">
            Copy this invite code now. You won't see it again.
          </Text.Text>
          <Code.Snippet label="Invite code" code={[[token]]} />
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
