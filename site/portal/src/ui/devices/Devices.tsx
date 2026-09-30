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

import { type Activation, type License } from "@/server/db/schema";
import { type Machine } from "@/server/license/desktop";
import { post, reload } from "@/ui/api";
import { date, machineName, statusOf } from "@/ui/format";
import { RenameDialog } from "@/ui/licenses/RenameDialog";
import { StatusTag } from "@/ui/licenses/StatusTag";
import * as Modal from "@/ui/Modal";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

export interface DevicesProps {
  /** machines are the machines signed in from the Desktop app that hold a seat. */
  machines: Machine[];
  now: Date;
}

const COLUMNS = "minmax(0, 2fr) 14rem 16rem 16rem 18rem";

/** validity says how much longer a machine keeps working. */
const validity = (lic: License, now: Date): string =>
  statusOf(lic, now) === "expired"
    ? `Expired ${date(lic.expiresAt)}. Open the app on that machine to renew.`
    : `Valid until ${date(lic.expiresAt)}`;

/** Devices lists the machines a personal user signed in from Synnax Desktop. */
export const Devices = ({ machines, now }: DevicesProps): ReactElement => (
  <Page title="Devices" subtitle="Machines signed in from Synnax Desktop">
    {machines.length === 0 ? (
      <Empty
        message="No machines yet"
        description="Sign in from the Synnax Desktop app and this machine appears here."
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
            <Text.Text level="p" color={10}>
              {date(a.firstSeen)}
            </Text.Text>
            <Text.Text level="p" color={10}>
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
              <UnlinkDialog activation={a} />
            </Flex.Box>
          </Row>
        ))}
      </Table>
    )}
  </Page>
);

const UnlinkDialog = ({ activation }: { activation: Activation }): ReactElement => (
  <Modal.Frame
    name="Unlink this device"
    icon={<Icon.Disconnect />}
    trigger={
      <Dialog.Trigger variant="text" size="small" hideCaret status="error">
        Unlink
      </Dialog.Trigger>
    }
  >
    <UnlinkContent activation={activation} />
  </Modal.Frame>
);

const UnlinkContent = ({ activation }: { activation: Activation }): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(async () => {
    await post(`/api/activations/${activation.key}/release`);
    close();
    reload();
  });
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Unlink {machineName(activation)}?
        </Text.Text>
        <Text.Text level="p" color={10}>
          The Desktop app on that machine stops renewing its license and asks you to
          sign in again.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={1000}
        >
          Unlink
        </Button.Button>
      </Modal.Footer>
    </>
  );
};
