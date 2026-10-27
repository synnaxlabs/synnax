// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { ADMIN_ROLE } from "@/server/directory";
import { type Invited } from "@/server/invitation";
import { scoped } from "@/shell";
import { navigate, post, reload } from "@/ui/api";
import { Auth } from "@/ui/auth";
import { useClerk } from "@/ui/clerk";
import { Tile } from "@/ui/Tile";
import { useAction } from "@/ui/useAction";

export interface Viewer {
  /** email is the address the visitor is logged in with. */
  email: string;
  /** addresses are the visitor's verified email addresses, in lowercase. */
  addresses: string[];
}

export interface AcceptProps {
  /** invited is the invitation the link opens, or null when it is closed. */
  invited: Invited | null;
  /** viewer is the logged-in visitor, or null for one who is logged out. */
  viewer: Viewer | null;
  now: Date;
}

const invitedAs = (role: string): string =>
  role === ADMIN_ROLE ? "an admin" : "a member";

// Reloading after the log out shows the page to a logged-out visitor.
const useLogOut = (): (() => void) | undefined => {
  const clerk = useClerk();
  if (clerk == null) return undefined;
  return () => {
    void clerk.signOut(() => {
      reload();
      return Promise.resolve();
    });
  };
};

const Closed = ({
  title,
  description,
}: Pick<Auth.CardProps, "title" | "description">): ReactElement => (
  <Auth.Card
    icon={
      <Tile>
        <Icon.Warning />
      </Tile>
    }
    title={title}
    description={description}
    footer={
      <Button.Button variant="text" size="small" textColor={9} href="/">
        Go to the portal
        <Icon.Arrow.Right />
      </Button.Button>
    }
  />
);

const Guest = ({ invitation: inv, team }: Invited): ReactElement => {
  const path = `/invitations/${inv.key}`;
  return (
    <Auth.Card
      title={`Join ${team.name}`}
      description={
        `You are invited as ${invitedAs(inv.role)}. Log in or create an account ` +
        `with ${inv.email} to accept.`
      }
    >
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        href={Auth.withTarget("/sign-up", path)}
      >
        Create an account
      </Button.Button>
      <Button.Button
        variant="outlined"
        size="large"
        full="x"
        justify="center"
        href={Auth.withTarget("/login", path)}
      >
        Log in
      </Button.Button>
    </Auth.Card>
  );
};

interface ViewedProps {
  invited: Invited;
  viewer: Viewer;
}

const Mismatch = ({
  invited: { invitation: inv, team },
  viewer,
}: ViewedProps): ReactElement => {
  const logOut = useLogOut();
  return (
    <Auth.Card
      title={`Join ${team.name}`}
      description={
        `This invitation is for ${inv.email}, but you are logged in as ` +
        `${viewer.email}.`
      }
    >
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        onClick={logOut}
        disabled={logOut == null}
      >
        Log out
      </Button.Button>
    </Auth.Card>
  );
};

const Join = ({
  invited: { invitation: inv, team },
  viewer,
}: ViewedProps): ReactElement => {
  const logOut = useLogOut();
  const join = useAction(async () => {
    await post(`/api/invitations/${inv.key}/accept`);
    navigate(scoped("/", team));
  });
  return (
    <Auth.Card
      title={`Join ${team.name}`}
      description={`You are invited as ${invitedAs(inv.role)}.`}
      error={join.error}
      footer={
        <Text.Text level="small" color={9}>
          Logged in as {viewer.email}.{" "}
          <Text.Text el="button" level="small" variant="link" onClick={logOut}>
            Log out
          </Text.Text>
        </Text.Text>
      }
    >
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        onClick={join.run}
        status={join.loading ? "loading" : undefined}
        trigger={["Enter"]}
      >
        Join {team.name}
        <Icon.Arrow.Right />
      </Button.Button>
    </Auth.Card>
  );
};

/**
 * Accept is the page an invitation email links to. It joins the team for a visitor
 * logged in with the invited address, and asks anyone else to log in with it.
 */
export const Accept = ({ invited, viewer, now }: AcceptProps): ReactElement => {
  if (invited == null)
    return (
      <Closed
        title="This invitation is closed"
        description="It was accepted or revoked. Ask a team admin to send a new one."
      />
    );
  const { invitation: inv, team } = invited;
  if (inv.expiresAt <= now)
    return (
      <Closed
        title="This invitation expired"
        description={`Ask an admin of ${team.name} to send it again.`}
      />
    );
  if (viewer == null) return <Guest {...invited} />;
  if (!viewer.addresses.includes(inv.email))
    return <Mismatch invited={invited} viewer={viewer} />;
  return <Join invited={invited} viewer={viewer} />;
};
