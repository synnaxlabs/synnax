// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type License, type Organization } from "@/server/db/schema";
import { term } from "@/server/license/limits";
import { date, statusOf } from "@/ui/format";
import { Licenses } from "@/ui/licenses";
import { Row, Table } from "@/ui/Table";

export interface LicenseEntry {
  license: License;
  seats: number;
  /** organization shows in its own column when set. */
  organization?: Organization;
}

export interface LicenseTableProps {
  entries: LicenseEntry[];
  now: Date;
}

const WITH_ORGANIZATION =
  "minmax(0, 2fr) minmax(0, 1.5fr) 14rem minmax(0, 1.4fr) 10rem 14rem 3rem";
const WITHOUT_ORGANIZATION = "minmax(0, 2fr) 14rem minmax(0, 1.4fr) 10rem 14rem 3rem";

/** LicenseTable lists licenses for staff, each row opening the license. */
export const LicenseTable = ({ entries, now }: LicenseTableProps): ReactElement => {
  const owned = entries.some((e) => e.organization != null);
  const columns = owned ? WITH_ORGANIZATION : WITHOUT_ORGANIZATION;
  const head = ["Label", "Status", "Term", "Machines", "Issued", ""];
  if (owned) head.splice(1, 0, "Organization");
  return (
    <Table columns={columns} head={head}>
      {entries.map(({ license: lic, seats, organization: org }) => (
        <Row key={lic.key} columns={columns} href={`/licenses/${lic.key}`}>
          <Text.Text level="p" weight={500} overflow="fade">
            {lic.label}
          </Text.Text>
          {owned && (
            <Text.Text level="p" color={9} overflow="fade">
              {org?.name}
            </Text.Text>
          )}
          <Flex.Box>
            <Licenses.StatusTag status={statusOf(lic, now)} />
          </Flex.Box>
          <Text.Text level="p" color={9} overflow="fade">
            {term(lic)}
          </Text.Text>
          <Text.Text level="p" color={9}>
            {seats} of {lic.nodes}
          </Text.Text>
          <Text.Text level="p" color={9}>
            {date(lic.issuedAt)}
          </Text.Text>
        </Row>
      ))}
    </Table>
  );
};
