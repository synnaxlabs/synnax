// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type License } from "@/server/db/schema";
import { type Machine } from "@/server/license/desktop";
import { LogOutDialog } from "@/ui/devices/LogOutDialog";
import { date, machineName, statusOf } from "@/ui/format";
import { RenameDialog } from "@/ui/licenses/RenameDialog";
import { StatusTag } from "@/ui/licenses/StatusTag";
import { DOWNLOAD_URL } from "@/ui/links";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface DevicesProps {
  /** machines are the machines logged in from the Desktop app that hold a seat. */
  machines: Machine[];
  now: Date;
}

const COLUMNS = "minmax(0, 2fr) 14rem 16rem 16rem 18rem";

/** validity says how much longer a machine keeps working. */
const validity = (lic: License, now: Date): string =>
  statusOf(lic, now) === "expired"
    ? `Expired ${date(lic.expiresAt)}. Open the app on that machine to renew.`
    : `Valid until ${date(lic.expiresAt)}`;

/** Devices lists the machines a personal user logged in from Synnax Desktop. */
export const Devices = ({ machines, now }: DevicesProps): ReactElement => (
  <Page title="Devices" subtitle="Machines logged in from Synnax Desktop">
    {machines.length === 0 ? (
      <Empty
        icon={<Icon.Computer />}
        message="No machines yet"
        description="Log in from the Synnax Desktop app and the machine appears here"
        action={
          <Button.Button variant="filled" href={DOWNLOAD_URL}>
            <Icon.Download />
            Download Synnax Desktop
          </Button.Button>
        }
      />
    ) : (
      <Table
        columns={COLUMNS}
        head={["Machine", "Status", "First seen", "Last renewal", ""]}
      >
        {machines.map(({ activation: a, license: lic }) => (
          <Row key={a.key} columns={COLUMNS}>
            <Flex.Box y gap="tiny" className="portal-list__stack">
              <Text.Text level="p" overflow="ellipsis">
                {machineName(a)}
              </Text.Text>
              <Text.Text level="small" color={9} overflow="ellipsis">
                {validity(lic, now)}
              </Text.Text>
            </Flex.Box>
            <Flex.Box>
              <StatusTag status={statusOf(lic, now)} />
            </Flex.Box>
            <Text.Text level="p" color={9}>
              {date(a.firstSeen)}
            </Text.Text>
            <Text.Text level="p" color={9}>
              {date(a.lastSeen)}
            </Text.Text>
            <Flex.Box x justify="end" gap="small">
              <RenameDialog
                activation={a}
                trigger={
                  <Dialog.Trigger variant="text" size="small" hideCaret>
                    Rename
                  </Dialog.Trigger>
                }
              />
              <LogOutDialog activation={a} />
            </Flex.Box>
          </Row>
        ))}
      </Table>
    )}
  </Page>
);
