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
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import {
  type Activation,
  type Event,
  type License as LicenseRecord,
  type Organization,
} from "@/server/db/schema";
import { deny } from "@/server/license/deny";
import { filename } from "@/server/license/limits";
import { post, postFile, reload, save } from "@/ui/api";
import { Fact, Facts } from "@/ui/Facts";
import {
  channels,
  date,
  dateTime,
  describeEvent,
  edition,
  machineName,
  statusOf,
  term,
} from "@/ui/format";
import { ActivateDialog } from "@/ui/licenses/ActivateDialog";
import { EditDialog } from "@/ui/licenses/EditDialog";
import { RenameDialog } from "@/ui/licenses/RenameDialog";
import { StatusTag } from "@/ui/licenses/StatusTag";
import * as Modal from "@/ui/Modal";
import { Empty, Page, Section } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

export interface LicenseProps {
  license: LicenseRecord;
  organization: Organization;
  activations: Activation[];
  events: Event[];
  /** actors is the name each Clerk user id in `events` reads as. */
  actors: Record<string, string>;
  staff: boolean;
  now: Date;
}

const MACHINE_COLUMNS = "minmax(0, 2fr) 12rem 12rem 6rem";

/** License shows one license: its terms, the machines holding seats, and its history. */
export const License = ({
  license: lic,
  organization,
  activations,
  events,
  actors,
  staff,
  now,
}: LicenseProps): ReactElement => {
  const status = statusOf(lic, now);
  const held = activations.filter((a) => a.releasedAt == null);
  const released = activations.filter((a) => a.releasedAt != null);
  const machines = Object.fromEntries(activations.map((a) => [a.key, machineName(a)]));
  return (
    <Page
      title={lic.label}
      subtitle={
        <Flex.Box x align="center" gap="small">
          <StatusTag status={status} />
          <Text.Text level="p" color={9}>
            {organization.name}
          </Text.Text>
        </Flex.Box>
      }
      actions={
        <>
          {staff && <StaffActions license={lic} now={now} />}
          {deny(lic, now) == null && (
            <ActivateDialog licenseKey={lic.key} label={lic.label} />
          )}
        </>
      }
    >
      <LicenseFacts license={lic} held={held.length} />
      <Section title="Machines">
        {held.length === 0 ? (
          <Empty
            icon={<Icon.Computer />}
            message="No machines hold a seat"
            description="Activate a machine to give a Core its license key."
          />
        ) : (
          <Table
            columns={MACHINE_COLUMNS}
            head={["Machine", "First seen", "Last seen", ""]}
          >
            {held.map((a) => (
              <Row key={a.key} columns={MACHINE_COLUMNS}>
                <Text.Text level="p" overflow="ellipsis">
                  {machineName(a)}
                </Text.Text>
                <Text.Text level="p" color={9}>
                  {date(a.firstSeen)}
                </Text.Text>
                <Text.Text level="p" color={9}>
                  {date(a.lastSeen)}
                </Text.Text>
                <Flex.Box justify="end">
                  <MachineMenu activation={a} label={lic.label} />
                </Flex.Box>
              </Row>
            ))}
          </Table>
        )}
      </Section>
      {released.length > 0 && (
        <Section title="Released">
          <Table
            columns={MACHINE_COLUMNS}
            head={["Machine", "First seen", "Last seen", "Released"]}
          >
            {released.map((a) => (
              <Row key={a.key} columns={MACHINE_COLUMNS}>
                <Text.Text level="p" color={9} overflow="ellipsis">
                  {machineName(a)}
                </Text.Text>
                <Text.Text level="p" color={9}>
                  {date(a.firstSeen)}
                </Text.Text>
                <Text.Text level="p" color={9}>
                  {date(a.lastSeen)}
                </Text.Text>
                <Text.Text level="p" color={9}>
                  {date(a.releasedAt)}
                </Text.Text>
              </Row>
            ))}
          </Table>
        </Section>
      )}
      <Section title="Activity">
        {events.length === 0 ? (
          <Empty icon={<Icon.Log />} message="Nothing yet" />
        ) : (
          <Flex.Box y gap="small">
            {events.map((e) => (
              <Flex.Box key={e.key} x gap="medium" align="start">
                <Text.Text level="small" color={9} className="portal-activity__time">
                  {dateTime(e.at)}
                </Text.Text>
                <Text.Text level="small" color={10}>
                  {describeEvent(e, { machines, actors })}
                </Text.Text>
              </Flex.Box>
            ))}
          </Flex.Box>
        )}
      </Section>
    </Page>
  );
};

const LicenseFacts = ({
  license: lic,
  held,
}: {
  license: LicenseRecord;
  held: number;
}): ReactElement => (
  <Facts>
    <Fact label="Edition" value={edition(lic.edition)} />
    <Fact label="Term" value={term(lic)} />
    <Fact label="Machines" value={`${held} of ${lic.nodes} seats in use`} />
    <Fact label="Channels per Core" value={channels(lic.channels)} />
    <Fact label="Issued" value={date(lic.issuedAt)} />
    {lic.revokedAt != null && <Fact label="Revoked" value={date(lic.revokedAt)} />}
    <Fact label="Key" value={lic.key} code />
  </Facts>
);

interface MachineMenuProps {
  activation: Activation;
  label: string;
}

const MachineMenu = ({ activation, label }: MachineMenuProps): ReactElement => {
  const [releasing, setReleasing] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const download = useAction(async () => {
    const blob = await postFile(`/api/activations/${activation.key}/download`);
    save(blob, filename(label));
  });
  const release = (): void => setReleasing(true);
  const rename = (): void => setRenaming(true);
  return (
    <>
      {download.error != null && (
        <Status.Summary variant="error" level="small" message={download.error} />
      )}
      <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
        <Dialog.Trigger
          variant="text"
          size="small"
          hideCaret
          aria-label="Machine actions"
        >
          <Icon.KebabMenu />
        </Dialog.Trigger>
        <Dialog.Dialog bordered rounded background={1} className="portal-machine-menu">
          <Menu.Menu
            level="small"
            onChange={{ download: download.run, rename, release }}
          >
            <Menu.Item itemKey="download">
              <Icon.Download />
              Download license key
            </Menu.Item>
            <Menu.Item itemKey="rename">
              <Icon.Rename />
              Rename
            </Menu.Item>
            <Menu.Item itemKey="release" status="error">
              <Icon.Release />
              Release seat
            </Menu.Item>
          </Menu.Menu>
        </Dialog.Dialog>
      </Dialog.Frame>
      <Modal.Frame
        name="Release this seat"
        icon={<Icon.Release />}
        visible={releasing}
        onVisibleChange={setReleasing}
      >
        <ReleaseContent activation={activation} />
      </Modal.Frame>
      <RenameDialog
        activation={activation}
        visible={renaming}
        onVisibleChange={setRenaming}
      />
    </>
  );
};

const ReleaseContent = ({ activation }: { activation: Activation }): ReactElement => {
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
          Release the seat held by {machineName(activation)}?
        </Text.Text>
        <Text.Text level="p" color={9}>
          The Core on that machine loses its license at its next check. Activate it
          again to give it a new license key.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error} hint="Press and hold Release to confirm">
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={1000}
        >
          Release
        </Button.Button>
      </Modal.Footer>
    </>
  );
};

interface StaffActionsProps {
  license: LicenseRecord;
  now: Date;
}

const StaffActions = ({ license: lic, now }: StaffActionsProps): ReactElement => {
  const floating = useAction(async () => {
    const blob = await postFile(`/api/licenses/${lic.key}/floating`);
    save(blob, filename(lic.label));
  });
  return (
    <>
      {floating.error != null && (
        <Status.Summary variant="error" level="small" message={floating.error} />
      )}
      {lic.revokedAt == null && <EditDialog license={lic} />}
      {deny(lic, now) == null && (
        <Button.Button
          variant="outlined"
          onClick={floating.run}
          status={floating.loading ? "loading" : undefined}
          tooltip="Download a license key bound to no machine, for CI runners"
        >
          <Icon.Download />
          Floating license key
        </Button.Button>
      )}
      {lic.revokedAt == null && (
        <Modal.Frame
          name={`${lic.label || "License"}.Revoke`}
          icon={<Icon.Delete />}
          trigger={
            <Dialog.Trigger variant="outlined" status="error" hideCaret>
              <Icon.Delete />
              Revoke
            </Dialog.Trigger>
          }
        >
          <RevokeContent license={lic} />
        </Modal.Frame>
      )}
    </>
  );
};

const RevokeContent = ({ license: lic }: { license: LicenseRecord }): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(async () => {
    await post(`/api/licenses/${lic.key}/revoke`);
    close();
    reload();
  });
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Revoke "{lic.label}"?
        </Text.Text>
        <Text.Text level="p" color={9}>
          Every Core running under it loses its license at its next check. The
          organization's admins are emailed. There is no undo.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error} hint="Press and hold Revoke to confirm">
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={1500}
        >
          Revoke
        </Button.Button>
      </Modal.Footer>
    </>
  );
};
