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

import { date, statusOf, term } from "@/portal/ui/format";
import { StatusTag } from "@/portal/ui/licenses/StatusTag";
import { Empty, Page } from "@/portal/ui/Page";
import { IssueDialog } from "@/portal/ui/staff/IssueDialog";
import { Row, Table } from "@/portal/ui/Table";
import { type License, type Organization } from "@/server/db/schema";
import { type Listed } from "@/server/directory";

export interface StaffLicenseRow {
  license: License;
  organization: Organization;
}

export interface StaffLicensesProps {
  /** teams are the organizations in Clerk a license can be issued to. */
  teams: Listed[];
  licenses: StaffLicenseRow[];
  now: Date | string;
}

const COLUMNS =
  "minmax(0, 2fr) minmax(0, 1.5fr) 10rem minmax(0, 1.4fr) 8rem 10rem 3rem";

/** StaffLicenses lists every license and issues new ones. Staff only. */
export const StaffLicenses = ({
  teams,
  licenses,
  now,
}: StaffLicensesProps): ReactElement => {
  const at = new Date(now);
  return (
    <Page
      title="All licenses"
      subtitle="Every license issued, newest first"
      actions={<IssueDialog teams={teams} />}
    >
      {licenses.length === 0 ? (
        <Empty message="No licenses issued yet" />
      ) : (
        <Table
          columns={COLUMNS}
          head={["Label", "Organization", "Status", "Term", "Machines", "Issued", ""]}
        >
          {licenses.map(({ license: lic, organization: org }) => (
            <Row key={lic.key} columns={COLUMNS} href={`/portal/licenses/${lic.key}`}>
              <Text.Text level="p" weight={500} overflow="ellipsis">
                {lic.label || "Untitled license"}
              </Text.Text>
              <Text.Text level="p" color={10} overflow="ellipsis">
                {org.name}
              </Text.Text>
              <Flex.Box>
                <StatusTag status={statusOf(lic, at)} />
              </Flex.Box>
              <Text.Text level="p" color={10} overflow="ellipsis">
                {term(lic)}
              </Text.Text>
              <Text.Text level="p" color={10}>
                {lic.nodes}
              </Text.Text>
              <Text.Text level="p" color={10}>
                {date(lic.issuedAt)}
              </Text.Text>
            </Row>
          ))}
        </Table>
      )}
    </Page>
  );
};
