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
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Organization } from "@/server/db/schema";
import { deny } from "@/server/license/deny";
import { type Held } from "@/server/license/list";
import { channels, date, edition, statusOf, term } from "@/ui/format";
import { ActivateDialog } from "@/ui/licenses/ActivateDialog";
import { StatusTag } from "@/ui/licenses/StatusTag";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface LicensesProps {
  organization: Organization;
  licenses: Held[];
  now: Date;
}

const COLUMNS = "minmax(0, 2fr) 14rem 10rem minmax(0, 1.4fr) 10rem 3rem";

/** Licenses lists an organization's licenses, one row per license. */
export const Licenses = ({
  organization,
  licenses,
  now,
}: LicensesProps): ReactElement => {
  const activatable = licenses.filter((l) => deny(l.license, now) == null);
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
          icon={<Icon.Access />}
          message="No licenses yet"
          description="Synnax Labs issues licenses. Contact us to ask for one."
        />
      ) : (
        <Table
          columns={COLUMNS}
          head={["Label", "Status", "Machines", "Term", "Issued", ""]}
        >
          {licenses.map(({ license: lic, seats }) => (
            <Row key={lic.key} columns={COLUMNS} href={`/licenses/${lic.key}`}>
              <Flex.Box y gap="tiny" className="portal-list__stack">
                <Text.Text level="p" weight={500} overflow="ellipsis">
                  {lic.label}
                </Text.Text>
                <Text.Text level="small" color={9}>
                  {edition(lic.edition)}, {channels(lic.channels)} channels
                </Text.Text>
              </Flex.Box>
              <Flex.Box>
                <StatusTag status={statusOf(lic, now)} />
              </Flex.Box>
              <Text.Text level="p" color={9}>
                {seats} of {lic.nodes}
              </Text.Text>
              <Text.Text level="p" color={9} overflow="ellipsis">
                {term(lic)}
              </Text.Text>
              <Text.Text level="p" color={9}>
                {date(lic.issuedAt)}
              </Text.Text>
            </Row>
          ))}
        </Table>
      )}
    </Page>
  );
};
