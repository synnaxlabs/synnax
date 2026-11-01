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
import { dateTime, describeEvent, machineName } from "@/ui/format";
import { Empty } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface ActivityTableProps extends Activity {
  /** organizations shows each event's organization in its own column. */
  organizations?: boolean;
  /** filtered says the page is filtered, for the message when it is empty. */
  filtered?: boolean;
}

const WITH_ORGANIZATION = "24rem minmax(0, 1.2fr) minmax(0, 1.2fr) minmax(0, 3fr) 3rem";
const WITHOUT_ORGANIZATION = "24rem minmax(0, 1.2fr) minmax(0, 3fr) 3rem";

/** ActivityTable lists audit events, each linking to the license it concerns. */
export const ActivityTable = ({
  page,
  actors,
  olderHref,
  organizations = false,
  filtered = false,
}: ActivityTableProps): ReactElement => {
  if (page.entries.length === 0)
    return (
      <Empty
        icon={<Icon.Log />}
        message={filtered ? "No activity matches" : "No activity yet"}
      />
    );
  const machines = Object.fromEntries(
    page.entries.flatMap(({ event: e, machine }) =>
      e.activation == null || machine == null
        ? []
        : [[e.activation, machineName(machine)]],
    ),
  );
  const columns = organizations ? WITH_ORGANIZATION : WITHOUT_ORGANIZATION;
  const head = ["Time", "License", "Event", ""];
  if (organizations) head.splice(1, 0, "Organization");
  return (
    <Flex.Box y gap="medium" full="x">
      <Table columns={columns} head={head}>
        {page.entries.map(({ event: e, organization, license }) => (
          <Row
            key={e.key}
            columns={columns}
            href={
              e.license != null
                ? `/licenses/${e.license}`
                : organizations && e.organization != null
                  ? `/admin/organizations/${e.organization}`
                  : undefined
            }
          >
            <Text.Text level="p" color={9}>
              {dateTime(e.at)}
            </Text.Text>
            {organizations && (
              <Text.Text level="p" color={10} overflow="fade">
                {organization}
              </Text.Text>
            )}
            <Text.Text level="p" color={10} overflow="fade">
              {license}
            </Text.Text>
            <Text.Text level="p" color={11} overflow="fade">
              {describeEvent(e, { machines, actors })}
            </Text.Text>
          </Row>
        ))}
      </Table>
      {olderHref != null && (
        <Flex.Box x justify="center">
          <Button.Button variant="outlined" href={olderHref}>
            Older events
          </Button.Button>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
