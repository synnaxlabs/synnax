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
import { date, shortHash } from "@/portal/ui/format";
import * as Modal from "@/portal/ui/Modal";
import { Empty, Page } from "@/portal/ui/Page";
import { Row, Table } from "@/portal/ui/Table";
import { useAction } from "@/portal/ui/useAction";
import { type Activation } from "@/server/db/schema";

export interface DesktopProps {
  /** devices are the machines signed in from the Desktop app that hold a seat. */
  devices: Activation[];
}

const COLUMNS = "minmax(0, 2fr) 12rem 12rem 12rem";

/** Desktop is a personal user's portal home: their Desktop devices. */
export const Desktop = ({ devices }: DesktopProps): ReactElement => (
  <Page title="Desktop" subtitle="Machines signed in from the Synnax Desktop app">
    {devices.length === 0 ? (
      <Empty
        message="No devices yet"
        description="Sign in from the Synnax Desktop app and this machine appears here."
      />
    ) : (
      <Table columns={COLUMNS} head={["Machine", "First seen", "Last seen", ""]}>
        {devices.map((d) => (
          <Row key={d.key} columns={COLUMNS}>
            <Text.Text level="p" variant="code" overflow="ellipsis">
              {shortHash(d.fingerprint)}
            </Text.Text>
            <Text.Text level="p" color={10}>
              {date(d.firstSeen)}
            </Text.Text>
            <Text.Text level="p" color={10}>
              {date(d.lastSeen)}
            </Text.Text>
            <Flex.Box justify="end">
              <UnlinkDialog device={d} />
            </Flex.Box>
          </Row>
        ))}
      </Table>
    )}
    <Enterprise />
  </Page>
);

const UnlinkDialog = ({ device }: { device: Activation }): ReactElement => (
  <Modal.Frame
    name="Unlink this device"
    icon={<Icon.Disconnect />}
    trigger={
      <Dialog.Trigger variant="text" size="small" hideCaret status="error">
        Unlink
      </Dialog.Trigger>
    }
  >
    <UnlinkContent device={device} />
  </Modal.Frame>
);

const UnlinkContent = ({ device }: { device: Activation }): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(
    useCallback(async () => {
      await post(`/api/portal/activations/${device.key}/release`);
      close();
      await reload();
    }, [device.key, close]),
  );
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Unlink {shortHash(device.fingerprint)}?
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
