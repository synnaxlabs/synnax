// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Invitee } from "@/page";
import { reload } from "@/ui/api";
import { Card } from "@/ui/auth/Card";
import { useClerk } from "@/ui/clerk";
import { Invitations } from "@/ui/invitations";

/**
 * Invited shows a logged-in visitor who opened an invitation link their open
 * invitations, and a log out for one sent to a different email.
 */
export const Invited = ({ email, invitations }: Invitee): ReactElement => {
  const clerk = useClerk();
  // Reloading after the log out opens the invitation link as a logged-out visitor.
  const logOut = (): void => {
    void clerk?.signOut(() => {
      reload();
      return Promise.resolve();
    });
  };
  if (invitations.length === 0)
    return (
      <Card
        title="Join your team"
        description={
          <>
            No pending invitation for {email}. Log out to accept one sent to a different
            email.
          </>
        }
        footer={
          <Text.Text el="a" level="small" variant="link" href="/">
            Go to the portal
          </Text.Text>
        }
      >
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={logOut}
          disabled={clerk == null}
        >
          Log out
        </Button.Button>
      </Card>
    );
  return (
    <Card
      title="Join your team"
      description={`You are logged in as ${email}`}
      footer={
        <Text.Text level="small" color={9}>
          Sent to a different email?{" "}
          <Text.Text el="button" level="small" variant="link" onClick={logOut}>
            Log out
          </Text.Text>
        </Text.Text>
      }
    >
      <Invitations.List invitations={invitations} />
    </Card>
  );
};
