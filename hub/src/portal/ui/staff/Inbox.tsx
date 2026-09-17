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
import { ThreadStatus } from "@/portal/ui/support/ThreadStatus";
import { Row, Table } from "@/portal/ui/Table";
import { type Listed } from "@/server/support/thread";

export interface InboxThread extends Listed {
  organizationName: string;
}

export interface InboxProps {
  threads: InboxThread[];
}

const COLUMNS = "minmax(0, 3fr) minmax(0, 1.5fr) 10rem 10rem 3rem";

/** Inbox lists every support thread for staff, newest activity first. */
export const Inbox = ({ threads }: InboxProps): ReactElement => (
  <Page title="Inbox" subtitle="Every support thread, across organizations">
    {threads.length === 0 ? (
      <Empty message="No threads" />
    ) : (
      <Table
        columns={COLUMNS}
        head={["Title", "Organization", "Status", "Last message", ""]}
      >
        {threads.map((t) => (
          <Row key={t.key} columns={COLUMNS} href={`/portal/support/${t.key}`}>
            <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
              <Text.Text level="p" weight={500} overflow="ellipsis">
                {t.title}
              </Text.Text>
              <Text.Text level="small" color={9}>
                {t.issueIdentifier}
                {t.kind === "feedback" ? ", feedback" : ""}
              </Text.Text>
            </Flex.Box>
            <Text.Text level="p" color={10} overflow="ellipsis">
              {t.organizationName}
            </Text.Text>
            <Flex.Box>
              <ThreadStatus status={t.status} />
            </Flex.Box>
            <Text.Text level="p" color={10}>
              {date(t.updatedAt)}
            </Text.Text>
          </Row>
        ))}
      </Table>
    )}
  </Page>
);
