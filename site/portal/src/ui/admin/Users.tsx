// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import { isActive, MONTH, type Period, PERIODS, type Usage } from "@/server/usage";
import { type Entry } from "@/server/user";
import { userPath } from "@/shell";
import { ActiveTag } from "@/ui/admin/ActiveTag";
import { Fact, Facts } from "@/ui/Facts";
import { date } from "@/ui/format";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface UsersProps {
  entries: Entry[];
  now: Date;
}

const COLUMNS = "minmax(0, 2fr) 16rem 16rem 12rem 12rem 3rem";

const PERIOD_LABELS: Record<Period, string> = {
  day: "Today",
  week: "This week",
  month: "This month",
};

const ACTIVE_LABELS: Record<Period, string> = {
  day: "Daily active users",
  week: "Weekly active users",
  month: "Monthly active users",
};

const lastActive = ({ lastActive: t }: Usage): number => t?.getTime() ?? -Infinity;

/** Users lists every account by when it last used Synnax Desktop. Staff only. */
export const Users = ({ entries, now }: UsersProps): ReactElement => {
  const [search, setSearch] = useState("");
  const [period, setPeriod] = useState<Period | null>(null);
  const query = search.trim().toLowerCase();
  const shown = entries
    .filter(({ account: a, usage }) => {
      if (period != null && !isActive(usage, period, now)) return false;
      return `${a.name} ${a.email} ${a.userID}`.toLowerCase().includes(query);
    })
    .sort((a, b) => lastActive(b.usage) - lastActive(a.usage));
  return (
    <Page title="Users" subtitle="Every account, the most recently active first">
      {entries.length === 0 ? (
        <Empty icon={<Icon.User />} message="No users yet" />
      ) : (
        <>
          <Facts>
            <Fact label="Accounts" value={String(entries.length)} />
            {PERIODS.map((p) => (
              <Fact
                key={p}
                label={ACTIVE_LABELS[p]}
                value={String(entries.filter((e) => isActive(e.usage, p, now)).length)}
              />
            ))}
          </Facts>
          <Flex.Box x justify="between" align="center" gap="medium" full="x">
            <Input.Text
              value={search}
              onChange={setSearch}
              placeholder={
                <>
                  <Icon.Search />
                  Name, email, or user ID
                </>
              }
              className="portal-search"
            />
            <Select.Buttons<Period>
              value={period ?? undefined}
              onChange={setPeriod}
              allowNone
            >
              {PERIODS.map((p) => (
                <Select.Item key={p} itemKey={p} size="small">
                  {PERIOD_LABELS[p]}
                </Select.Item>
              ))}
            </Select.Buttons>
          </Flex.Box>
          {shown.length === 0 ? (
            <Empty icon={<Icon.Search />} message="No users match" />
          ) : (
            <Table
              columns={COLUMNS}
              head={[
                "User",
                "Last Desktop login",
                "Last active",
                "Active days",
                "Joined",
                "",
              ]}
            >
              {shown.map(({ account: a, usage }) => (
                <Row key={a.userID} columns={COLUMNS} href={userPath(a.userID)}>
                  <Flex.Box y gap="tiny" className="portal-list__stack">
                    <Text.Text level="p" weight={500} overflow="fade">
                      {a.name}
                    </Text.Text>
                    <Text.Text level="small" color={9} overflow="fade">
                      {a.email}
                    </Text.Text>
                  </Flex.Box>
                  <Text.Text level="p" color={9}>
                    {date(usage.lastLogin)}
                  </Text.Text>
                  {isActive(usage, "day", now) ? (
                    <Flex.Box>
                      <ActiveTag />
                    </Flex.Box>
                  ) : (
                    <Text.Text level="p" color={9}>
                      {date(usage.lastActive)}
                    </Text.Text>
                  )}
                  <Text.Text level="p" color={9}>
                    {usage.days === 0 ? "" : `${usage.days} of ${MONTH}`}
                  </Text.Text>
                  <Text.Text level="p" color={9}>
                    {date(a.joinedAt)}
                  </Text.Text>
                </Row>
              ))}
            </Table>
          )}
        </>
      )}
    </Page>
  );
};
