// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import { type Summary } from "@/server/organization";
import { RegisterDialog } from "@/ui/admin/RegisterDialog";
import { date, standing } from "@/ui/format";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface OrganizationsProps {
  teams: Summary[];
  now: Date;
}

const COLUMNS = "minmax(0, 2fr) 14rem 20rem 12rem 14rem 3rem";

// Counts name their noun, since phones show rows without the header.
const count = (n: number, noun: string): string =>
  `${n} ${n === 1 ? noun : `${noun}s`}`;

/** Organizations lists every team and registers new ones. Staff only. */
export const Organizations = ({ teams, now }: OrganizationsProps): ReactElement => {
  const [search, setSearch] = useState("");
  const query = search.trim().toLowerCase();
  const shown = teams
    .filter((t) => t.organization.name.toLowerCase().includes(query))
    .sort((a, b) => a.organization.name.localeCompare(b.organization.name));
  return (
    <Page
      title="Organizations"
      subtitle="Every team that can hold a license"
      actions={<RegisterDialog />}
    >
      {teams.length === 0 ? (
        <Empty icon={<Icon.Role />} message="No organizations yet" />
      ) : (
        <>
          <Input.Text
            value={search}
            onChange={setSearch}
            placeholder={
              <>
                <Icon.Search />
                Name
              </>
            }
            className="portal-search"
          />
          {shown.length === 0 ? (
            <Empty icon={<Icon.Search />} message="No organizations match" />
          ) : (
            <Table
              columns={COLUMNS}
              head={["Name", "Members", "Licenses", "Machines", "Next expiry", ""]}
            >
              {shown.map(({ organization: org, members, licenses }) => {
                const s = standing(licenses, now);
                return (
                  <Row
                    key={org.key}
                    columns={COLUMNS}
                    href={`/admin/organizations/${org.key}`}
                  >
                    <Text.Text level="p" weight={500} overflow="fade">
                      {org.name}
                    </Text.Text>
                    <Text.Text level="p" color={9}>
                      {count(members, "member")}
                    </Text.Text>
                    <Text.Text level="p" color={9}>
                      {count(s.active, "active license")}
                    </Text.Text>
                    <Text.Text level="p" color={9}>
                      {s.active === 0 ? "" : `${s.seats} of ${s.capacity}`}
                    </Text.Text>
                    <Text.Text level="p" color={9}>
                      {s.nextExpiry == null ? "" : date(s.nextExpiry)}
                    </Text.Text>
                  </Row>
                );
              })}
            </Table>
          )}
        </>
      )}
    </Page>
  );
};
