// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { date } from "@/portal/ui/format";
import { Empty, Page } from "@/portal/ui/Page";
import { NewThreadDialog } from "@/portal/ui/support/NewThreadDialog";
import { ThreadStatus } from "@/portal/ui/support/ThreadStatus";
import { Row, Table } from "@/portal/ui/Table";
import { type Organization } from "@/server/db/schema";
import { type Listed } from "@/server/support/thread";

export interface SupportProps {
  organization: Organization;
  threads: Listed[];
  staff: boolean;
}

const COLUMNS = "minmax(0, 3fr) 10rem 10rem 10rem 3rem";

/** Support lists an organization's threads and opens new ones. */
export const Support = ({
  organization,
  threads,
  staff,
}: SupportProps): ReactElement => (
  <Page
    title="Support"
    subtitle={organization.kind === "team" ? organization.name : undefined}
    actions={
      <NewThreadDialog
        organizationKey={organization.key}
        organizationName={organization.kind === "team" ? organization.name : undefined}
      />
    }
  >
    {threads.length === 0 ? (
      <Empty
        message="No threads yet"
        description="Open a thread to reach Synnax Labs. Replies land here and in your inbox."
      />
    ) : (
      <Table columns={COLUMNS} head={["Title", "Status", "Opened", "Last message", ""]}>
        {threads.map((t) => (
          <Row key={t.key} columns={COLUMNS} href={`/portal/support/${t.key}`}>
            <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
              <Text.Text level="p" weight={500} overflow="ellipsis">
                {t.title}
              </Text.Text>
              {staff && (
                <Text.Text level="small" color={9}>
                  {t.issueIdentifier}
                </Text.Text>
              )}
            </Flex.Box>
            <Flex.Box>
              <ThreadStatus status={t.status} />
            </Flex.Box>
            <Text.Text level="p" color={10}>
              {date(t.createdAt)}
            </Text.Text>
            <Text.Text level="p" color={10}>
              {date(t.updatedAt)}
            </Text.Text>
          </Row>
        ))}
      </Table>
    )}
  </Page>
);
