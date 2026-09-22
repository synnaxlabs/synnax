// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Flex, Icon, Menu, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { post, postFile, reload, save } from "@/portal/ui/api";
import {
  channels,
  date,
  dateTime,
  describeEvent,
  edition,
  type LicenseStatus,
  machineName,
  statusOf,
  term,
  usable,
} from "@/portal/ui/format";
import { ActivateDialog } from "@/portal/ui/licenses/ActivateDialog";
import { EditDialog } from "@/portal/ui/licenses/EditDialog";
import { RenameDialog } from "@/portal/ui/licenses/RenameDialog";
import { StatusTag } from "@/portal/ui/licenses/StatusTag";
import * as Modal from "@/portal/ui/Modal";
import { Empty, Page, Section } from "@/portal/ui/Page";
import { Row, Table } from "@/portal/ui/Table";
import { useAction } from "@/portal/ui/useAction";
import {
  type Activation,
  type Event,
  type License as LicenseRecord,
  type Organization,
} from "@/server/db/schema";

export interface LicenseProps {
  license: LicenseRecord;
  organization: Organization;
  activations: Activation[];
  events: Event[];
  /** actors is the name each Clerk user id in `events` reads as. */
  actors: Record<string, string>;
  staff: boolean;
  now: Date | string;
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
  const at = new Date(now);
  const status = statusOf(lic, at);
  const held = activations.filter((a) => a.releasedAt == null);
  const machines = Object.fromEntries(activations.map((a) => [a.key, machineName(a)]));
  return (
    <Page
      title={lic.label || "Untitled license"}
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
          {staff && <StaffActions license={lic} status={status} />}
          {usable(status) && <ActivateDialog licenseKey={lic.key} label={lic.label} />}
        </>
      }
    >
      <Facts license={lic} held={held.length} />
      <Section title="Machines">
        {held.length === 0 ? (
          <Empty
            message="No machines hold a seat"
            description="Activate a machine to give a Core its token."
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
                <Text.Text level="p" color={10}>
                  {date(a.firstSeen)}
                </Text.Text>
                <Text.Text level="p" color={10}>
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
      <Section title="Activity">
        {events.length === 0 ? (
          <Empty message="Nothing yet" />
        ) : (
          <Flex.Box y gap="small">
            {events.map((e) => (
              <Flex.Box key={e.key} x gap="medium" align="start">
                <Text.Text level="small" color={9} style={{ minWidth: "18rem" }}>
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

const Facts = ({
  license: lic,
  held,
}: {
  license: LicenseRecord;
  held: number;
}): ReactElement => (
  <Flex.Box bordered rounded background={1} style={{ padding: "3rem 4rem" }}>
    <Flex.Box className="portal-facts" full="x">
      <Fact label="Edition" value={edition(lic.edition)} />
      <Fact label="Term" value={term(lic)} />
      <Fact label="Machines" value={`${held} of ${lic.nodes} seats in use`} />
      <Fact label="Channels per Core" value={channels(lic.channels)} />
      <Fact label="Issued" value={date(lic.issuedAt)} />
      {lic.revokedAt != null && <Fact label="Revoked" value={date(lic.revokedAt)} />}
      <Fact label="Key" value={lic.key} code />
    </Flex.Box>
  </Flex.Box>
);

const Fact = ({
  label,
  value,
  code = false,
}: {
  label: string;
  value: string;
  code?: boolean;
}): ReactElement => (
  <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
    <Text.Text level="small" color={9}>
      {label}
    </Text.Text>
    <Text.Text level="p" variant={code ? "code" : "prose"} overflow="ellipsis">
      {value}
    </Text.Text>
  </Flex.Box>
);

interface MachineMenuProps {
  activation: Activation;
  label: string;
}

const MachineMenu = ({ activation, label }: MachineMenuProps): ReactElement => {
  const [releasing, setReleasing] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const download = useAction(
    useCallback(async () => {
      const blob = await postFile(`/api/portal/activations/${activation.key}/token`);
      save(blob, `${label || "synnax"}.license`);
    }, [activation.key, label]),
  );
  const release = useCallback(() => setReleasing(true), []);
  const rename = useCallback(() => setRenaming(true), []);
  return (
    <>
      <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
        <Dialog.Trigger
          variant="text"
          size="small"
          hideCaret
          aria-label="Machine actions"
        >
          <Icon.KebabMenu />
        </Dialog.Trigger>
        <Dialog.Dialog bordered rounded background={1} style={{ padding: "1rem" }}>
          <Menu.Menu
            level="small"
            onChange={{ download: download.run, rename, release }}
          >
            <Menu.Item itemKey="download">
              <Icon.Download />
              Download token
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
  const action = useAction(
    useCallback(async () => {
      await post(`/api/portal/activations/${activation.key}/release`);
      close();
      await reload();
    }, [activation.key, close]),
  );
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Release the seat held by {machineName(activation)}?
        </Text.Text>
        <Text.Text level="p" color={10}>
          The Core on that machine loses its license at its next check. Activate it
          again to give it a new token.
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
          Release
        </Button.Button>
      </Modal.Footer>
    </>
  );
};

interface StaffActionsProps {
  license: LicenseRecord;
  status: LicenseStatus;
}

const StaffActions = ({ license: lic, status }: StaffActionsProps): ReactElement => {
  const floating = useAction(
    useCallback(async () => {
      const blob = await postFile(`/api/portal/licenses/${lic.key}/floating`);
      save(blob, `${lic.label || "synnax"}.license`);
    }, [lic.key, lic.label]),
  );
  return (
    <>
      {status !== "revoked" && <EditDialog license={lic} />}
      {usable(status) && (
        <Button.Button
          variant="outlined"
          onClick={floating.run}
          status={floating.loading ? "loading" : undefined}
          tooltip="Download a token bound to no machine, for CI runners"
        >
          <Icon.Download />
          Floating token
        </Button.Button>
      )}
      {status !== "revoked" && (
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
  const action = useAction(
    useCallback(async () => {
      await post(`/api/portal/licenses/${lic.key}/revoke`);
      close();
      await reload();
    }, [lic.key, close]),
  );
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Revoke "{lic.label}"?
        </Text.Text>
        <Text.Text level="p" color={10}>
          Every Core running under it loses its license at its next check. The
          organization's admins are emailed. There is no undo.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={1500}
        >
          Hold to revoke
        </Button.Button>
      </Modal.Footer>
    </>
  );
};
