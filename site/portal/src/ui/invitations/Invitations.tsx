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

import { type Invited } from "@/server/invitation";
import { scoped } from "@/shell";
import { navigate, post } from "@/ui/api";
import { Section as PageSection } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

const COLUMNS = "minmax(0, 1fr) auto";

const Item = ({ invitation, team }: Invited): ReactElement => {
  const join = useAction(async () => {
    await post(`/api/invitations/${invitation.key}/accept`);
    navigate(scoped("/", team));
  });
  return (
    <Row columns={COLUMNS}>
      <Flex.Box y gap="tiny" className="portal-list__stack">
        <Text.Text level="p" weight={500} overflow="fade">
          {team.name}
        </Text.Text>
        {join.error != null && (
          <Status.Summary variant="error" level="small" message={join.error} />
        )}
      </Flex.Box>
      <Button.Button
        variant="filled"
        onClick={join.run}
        status={join.loading ? "loading" : undefined}
      >
        Join
      </Button.Button>
    </Row>
  );
};

export interface ListProps {
  /** invitations are the open invitations to the user's verified addresses. */
  invitations: Invited[];
}

/** List shows each team invitation with a button that joins the team. */
export const List = ({ invitations }: ListProps): ReactElement => (
  <Table columns={COLUMNS} head={["Team", ""]}>
    {invitations.map((i) => (
      <Item key={i.invitation.key} {...i} />
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
