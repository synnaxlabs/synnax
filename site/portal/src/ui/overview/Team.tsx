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

import { type Organization } from "@/server/db/schema";
import { type Held } from "@/server/license/list";
import { scoped } from "@/shell";
import { Fact, Facts } from "@/ui/Facts";
import { date, standing, statusOf } from "@/ui/format";
import { Licenses } from "@/ui/licenses";
import { Empty, Page, Section } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

/** RECENT is how many licenses the overview shows before linking to Licenses. */
const RECENT = 3;

const COLUMNS = "minmax(0, 2fr) 14rem 10rem";

export interface TeamProps {
  organization: Organization;
  /** licenses are the team's licenses, newest first. */
  licenses: Held[];
  now: Date;
}

/**
 * Team is the overview of a team: its license standing, its newest licenses, and the
 * way to its members.
 */
export const Team = ({ organization, licenses, now }: TeamProps): ReactElement => {
  const { active, seats, capacity, nextExpiry } = standing(licenses, now);
  return (
    <Page
      title={organization.name}
      subtitle="Synnax Enterprise"
      actions={
        active > 0 && (
          <Button.Button
            variant="filled"
            href={scoped("/licenses/activate", organization)}
          >
            <Icon.Add />
            Activate a machine
          </Button.Button>
        )
      }
    >
      <Facts>
        <Fact label="Active licenses" value={String(active)} />
        <Fact label="Machines" value={`${seats} of ${capacity} seats in use`} />
        <Fact
          label="Next expiry"
          value={nextExpiry == null ? "None" : date(nextExpiry)}
        />
      </Facts>
      <Section
        title="Licenses"
        actions={
          <Button.Button
            variant="text"
            textColor={9}
            href={scoped("/licenses", organization)}
          >
            View all
          </Button.Button>
        }
      >
        {licenses.length === 0 ? (
          <Empty
            icon={<Icon.Access />}
            message="No licenses yet"
            description="Synnax Labs issues licenses. Contact us to ask for one."
          />
        ) : (
          <Table columns={COLUMNS} head={["Label", "Status", "Machines"]}>
            {licenses.slice(0, RECENT).map(({ license: lic, seats: held }) => (
              <Row key={lic.key} columns={COLUMNS} href={`/licenses/${lic.key}`}>
                <Text.Text level="p" weight={500} overflow="ellipsis">
                  {lic.label}
                </Text.Text>
                <Flex.Box>
                  <Licenses.StatusTag status={statusOf(lic, now)} />
                </Flex.Box>
                <Text.Text level="p" color={9}>
                  {held} of {lic.nodes}
                </Text.Text>
              </Row>
            ))}
          </Table>
        )}
      </Section>
      <Section
        title="Members"
        actions={
          <Button.Button
            variant="text"
            textColor={9}
            href={scoped("/members", organization)}
          >
            Manage
          </Button.Button>
        }
      >
        <Text.Text level="p" color={9}>
          Every member can use the team's licenses. Admins invite members and change
          roles.
        </Text.Text>
      </Section>
    </Page>
  );
};
