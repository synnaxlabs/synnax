// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Invited } from "@/server/organization";
import { scoped } from "@/shell";
import { navigate } from "@/ui/api";
import { useClerk, type User } from "@/ui/clerk";
import { Section as PageSection } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

const COLUMNS = "minmax(0, 1fr) auto";

const PAGE = 100;

// Clerk accepts an invitation only through the object it returns for it.
const find = async (user: User, invitationID: string) => {
  for (let page = 1; ; page++) {
    const { data } = await user.getOrganizationInvitations({
      status: "pending",
      initialPage: page,
      pageSize: PAGE,
    });
    const found = data.find((i) => i.id === invitationID);
    if (found != null) return found;
    if (data.length < PAGE) throw new Error("This invitation is no longer open");
  }
};

const Item = ({ invitationID, team }: Invited): ReactElement => {
  const clerk = useClerk();
  const join = useAction(async () => {
    if (clerk?.user == null) return;
    await (await find(clerk.user, invitationID)).accept();
    navigate(scoped("/", team));
  });
  return (
    <Row columns={COLUMNS}>
      <Flex.Box y gap="tiny" className="portal-list__stack">
        <Text.Text level="p" weight={500} overflow="ellipsis">
          {team.name}
        </Text.Text>
        {join.error != null && (
          <Status.Summary variant="error" level="small" message={join.error} />
        )}
      </Flex.Box>
      <Button.Button
        variant="filled"
        onClick={join.run}
        disabled={clerk == null}
        status={join.loading ? "loading" : undefined}
      >
        Join
      </Button.Button>
    </Row>
  );
};

export interface ListProps {
  /** invitations are the team invitations the user has not accepted. */
  invitations: Invited[];
}

/** List shows each team invitation with a button that joins the team. */
export const List = ({ invitations }: ListProps): ReactElement => (
  <Table columns={COLUMNS} head={["Team", ""]}>
    {invitations.map((i) => (
      <Item key={i.invitationID} {...i} />
    ))}
  </Table>
);

/** Section is the overview's list of invitations. It renders nothing without any. */
export const Section = ({ invitations }: ListProps): ReactElement | null =>
  invitations.length === 0 ? null : (
    <PageSection title="Invitations">
      <List invitations={invitations} />
    </PageSection>
  );
