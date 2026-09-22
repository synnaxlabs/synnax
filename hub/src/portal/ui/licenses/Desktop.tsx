// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { post, reload } from "@/portal/ui/api";
import { Enterprise } from "@/portal/ui/Enterprise";
import { date, machineName, statusOf } from "@/portal/ui/format";
import { StatusTag } from "@/portal/ui/licenses/StatusTag";
import * as Modal from "@/portal/ui/Modal";
import { Empty, Page } from "@/portal/ui/Page";
import { Row, Table } from "@/portal/ui/Table";
import { useAction } from "@/portal/ui/useAction";
import { type Activation, type License } from "@/server/db/schema";
import { type Machine } from "@/server/license/desktop";

export interface DesktopProps {
  /** machines are the machines signed in from the Desktop app that hold a seat. */
  machines: Machine[];
  now: Date | string;
}

const COLUMNS = "minmax(0, 2fr) 10rem 12rem 12rem 12rem";

/** validity says how much longer a machine keeps working. */
const validity = (lic: License, at: Date): string =>
  statusOf(lic, at) === "expired"
    ? `Expired ${date(lic.expiresAt)}. Open the app on that machine to renew.`
    : `Valid until ${date(lic.expiresAt)}`;

/** Desktop is a personal user's portal home: their Desktop machines. */
export const Desktop = ({ machines, now }: DesktopProps): ReactElement => {
  const at = new Date(now);
  return (
    <Page title="Desktop" subtitle="Machines signed in from the Synnax Desktop app">
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
              <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
                <Text.Text level="p" overflow="ellipsis">
                  {machineName(a)}
                </Text.Text>
                <Text.Text level="small" color={9} overflow="ellipsis">
                  {validity(lic, at)}
                </Text.Text>
              </Flex.Box>
              <Flex.Box>
                <StatusTag status={statusOf(lic, at)} />
              </Flex.Box>
              <Text.Text level="p" color={10}>
                {date(a.firstSeen)}
              </Text.Text>
              <Text.Text level="p" color={10}>
                {date(a.lastSeen)}
              </Text.Text>
              <Flex.Box justify="end">
                <UnlinkDialog activation={a} />
              </Flex.Box>
            </Row>
          ))}
        </Table>
      )}
      <Enterprise />
    </Page>
  );
};

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
  const action = useAction(
    useCallback(async () => {
      await post(`/api/portal/activations/${activation.key}/unlink`);
      close();
      await reload();
    }, [activation.key, close]),
  );
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
