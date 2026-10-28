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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Activity } from "@/page";
import { personURL } from "@/server/analytics";
import { roleName } from "@/server/directory";
import { type Machine } from "@/server/license/desktop";
import { isActive, MONTH } from "@/server/usage";
import { type Detail, type Membership } from "@/server/user";
import { ActiveTag } from "@/ui/admin/ActiveTag";
import { ActivityTable } from "@/ui/admin/ActivityTable";
import { Fact, Facts } from "@/ui/Facts";
import { date, dateTime, machineName, statusOf } from "@/ui/format";
import { Licenses } from "@/ui/licenses";
import { Empty, Page, Section } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface UserProps extends Detail {
  /** activity is the audit log of the user's machines, or null with no machines. */
  activity: Activity | null;
  now: Date;
}

const MACHINE_COLUMNS = "minmax(0, 2fr) 14rem 16rem 16rem 3rem";
const TEAM_COLUMNS = "minmax(0, 2fr) 14rem 3rem";

const moment = (t: Date | null): string => (t == null ? "Never" : dateTime(t));

/** User is one account's staff page: its Desktop use, machines, teams, and activity. */
export const User = ({
  account,
  usage,
  machines,
  teams,
  activity,
  now,
}: UserProps): ReactElement => (
  <Page
    title={account.name}
    parent={{ label: "Users", href: "/admin/users" }}
    subtitle={
      <Flex.Box x align="center" gap="small">
        {account.email}
        {isActive(usage, "day", now) && <ActiveTag />}
      </Flex.Box>
    }
    actions={
      <Button.Button
        variant="outlined"
        href={personURL(account.userID)}
        target="_blank"
        rel="noreferrer"
      >
        <Icon.OpenExternal />
        Open in PostHog
      </Button.Button>
    }
  >
    <Facts>
      <Fact label="Last Desktop login" value={moment(usage.lastLogin)} />
      <Fact label="Last active" value={moment(usage.lastActive)} />
      <Fact label="Active days" value={`${usage.days} of the last ${MONTH}`} />
      <Fact label="Joined" value={date(account.joinedAt)} />
      <Fact label="User ID" value={account.userID} code />
    </Facts>
    <Section title="Machines">
      <Machines machines={machines} now={now} />
    </Section>
    <Section title="Teams">
      <Teams teams={teams} />
    </Section>
    {activity != null && (
      <Section title="Activity">
        <ActivityTable {...activity} />
      </Section>
    )}
  </Page>
);

const Machines = ({
  machines,
  now,
}: {
  machines: Machine[];
  now: Date;
}): ReactElement =>
  machines.length === 0 ? (
    <Empty icon={<Icon.Computer />} message="No machines logged in" />
  ) : (
    <Table
      columns={MACHINE_COLUMNS}
      head={["Machine", "Status", "First seen", "Last renewal", ""]}
    >
      {machines.map(({ activation: a, license: lic }) => (
        <Row key={a.key} columns={MACHINE_COLUMNS} href={`/licenses/${lic.key}`}>
          <Text.Text level="p" overflow="ellipsis">
            {machineName(a)}
          </Text.Text>
          <Flex.Box>
            <Licenses.StatusTag status={statusOf(lic, now)} />
          </Flex.Box>
          <Text.Text level="p" color={9}>
            {date(a.firstSeen)}
          </Text.Text>
          <Text.Text level="p" color={9}>
            {dateTime(a.lastSeen)}
          </Text.Text>
        </Row>
      ))}
    </Table>
  );

const Teams = ({ teams }: { teams: Membership[] }): ReactElement =>
  teams.length === 0 ? (
    <Empty icon={<Icon.Role />} message="No teams" />
  ) : (
    <Table columns={TEAM_COLUMNS} head={["Name", "Role", ""]}>
      {teams.map(({ organization: org, role }) => (
        <Row
          key={org.key}
          columns={TEAM_COLUMNS}
          href={`/admin/organizations/${org.key}`}
        >
          <Text.Text level="p" weight={500} overflow="ellipsis">
            {org.name}
          </Text.Text>
          <Text.Text level="p" color={9}>
            {roleName(role)}
          </Text.Text>
        </Row>
      ))}
    </Table>
  );
