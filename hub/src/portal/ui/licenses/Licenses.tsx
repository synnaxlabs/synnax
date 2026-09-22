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

import { channels, date, edition, statusOf, term, usable } from "@/portal/ui/format";
import { ActivateDialog } from "@/portal/ui/licenses/ActivateDialog";
import { StatusTag } from "@/portal/ui/licenses/StatusTag";
import { Empty, Page } from "@/portal/ui/Page";
import { Row, Table } from "@/portal/ui/Table";
import { type License, type Organization } from "@/server/db/schema";

export interface LicenseRow {
  license: License;
  seats: number;
}

export interface LicensesProps {
  organization: Organization;
  licenses: LicenseRow[];
  now: Date | string;
}

const COLUMNS = "minmax(0, 2fr) 10rem 10rem minmax(0, 1.4fr) 10rem 3rem";

/** Licenses lists an organization's licenses, one row per license. */
export const Licenses = ({
  organization,
  licenses,
  now,
}: LicensesProps): ReactElement => {
  const at = new Date(now);
  const activatable = licenses.filter((l) => usable(statusOf(l.license, at)));
  return (
    <Page
      title="Licenses"
      subtitle={organization.name}
      actions={
        activatable.length === 1 && (
          <ActivateDialog
            licenseKey={activatable[0].license.key}
            label={activatable[0].license.label}
          />
        )
      }
    >
      {licenses.length === 0 ? (
        <Empty
          message="No licenses yet"
          description="Synnax Labs issues licenses. Contact us to ask for one."
        />
      ) : (
        <Table
          columns={COLUMNS}
          head={["Label", "Status", "Machines", "Term", "Issued", ""]}
        >
          {licenses.map(({ license: lic, seats }) => (
            <Row key={lic.key} columns={COLUMNS} href={`/account/licenses/${lic.key}`}>
              <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
                <Text.Text level="p" weight={500} overflow="ellipsis">
                  {lic.label || "Untitled license"}
                </Text.Text>
                <Text.Text level="small" color={9}>
                  {edition(lic.edition)}, {channels(lic.channels)} channels
                </Text.Text>
              </Flex.Box>
              <Flex.Box>
                <StatusTag status={statusOf(lic, at)} />
              </Flex.Box>
              <Text.Text level="p" color={10}>
                {seats} of {lic.nodes}
              </Text.Text>
              <Text.Text level="p" color={10} overflow="ellipsis">
                {term(lic)}
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
