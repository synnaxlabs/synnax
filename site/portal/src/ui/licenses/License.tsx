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
import { channels, MAX_LABEL_LENGTH, term } from "@/server/license/limits";
import { activatePath } from "@/shell";
import { post } from "@/ui/api";
import { Fact, Facts } from "@/ui/Facts";
import {
  date,
  dateTime,
  describeEvent,
  edition,
  machineName,
  statusOf,
} from "@/ui/format";
import { MachineMenu } from "@/ui/licenses/MachineMenu";
import { StaffActions } from "@/ui/licenses/StaffActions";
import { StatusTag } from "@/ui/licenses/StatusTag";
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
  /** admin is true for staff and the admins of the owning team. */
  admin: boolean;
  now: Date;
}

const LABEL_ID = "license-label";

interface Renamed {
  event: (Omit<Event, "at"> & { at: string }) | null;
  actor: string;
}

const MACHINE_COLUMNS = "minmax(0, 2fr) 14rem 14rem 6rem";

/** License shows a license: its terms, the machines holding seats, and its history. */
export const License = ({
  license: stored,
  organization,
  activations,
  events: loadedEvents,
  actors: loadedActors,
  staff,
  admin,
  now,
}: LicenseProps): ReactElement => {
  const [label, setLabel] = useState(stored.label);
  const [events, setEvents] = useState(loadedEvents);
  const [actors, setActors] = useState(loadedActors);
  const [renaming, setRenaming] = useState(false);
  const lic = { ...stored, label };
  const rename = useAction(async (next: string) => {
    setLabel(next);
    try {
      const { event, actor } = await post<Renamed>(`/api/licenses/${lic.key}/label`, {
        label: next,
      });
      if (event == null) return;
      setEvents((prev) => [{ ...event, at: new Date(event.at) }, ...prev]);
      setActors((prev) => ({ ...prev, [event.actor]: actor }));
      document.title = document.title.replace(label, () => next);
    } catch (err) {
      setLabel(label);
      throw err as Error;
    }
  });
  const startRenaming = (): void => {
    setRenaming(true);
    Text.asyncEdit(LABEL_ID)
      .catch(console.error)
      .finally(() => setRenaming(false));
  };
  const status = statusOf(lic, now);
  const held = activations.filter((a) => a.releasedAt == null);
  const released = activations.filter((a) => a.releasedAt != null);
  const machines = Object.fromEntries(activations.map((a) => [a.key, machineName(a)]));
  const parent =
    staff && organization.kind === "team"
      ? { label: organization.name, href: `/admin/organizations/${organization.key}` }
      : undefined;
  return (
    <Page
      title={label}
      titleID={LABEL_ID}
      onTitleChange={renaming ? rename.run : undefined}
      titleMaxLength={MAX_LABEL_LENGTH}
      parent={parent}
      subtitle={
        <Flex.Box x align="center" gap="small">
          <StatusTag status={status} />
          {parent == null && (
            <Text.Text level="p" color={9}>
              {organization.name}
            </Text.Text>
          )}
        </Flex.Box>
      }
      actions={
        <>
          {admin && lic.edition === "enterprise" && (
            <Button.Button
              variant="outlined"
              onClick={startRenaming}
              disabled={rename.loading}
            >
              <Icon.Rename />
              Rename
            </Button.Button>
          )}
          {staff && (
            <StaffActions
              license={lic}
              now={now}
              back={parent?.href ?? "/admin/licenses"}
            />
          )}
          {deny(lic, now) == null && (
            <Button.Button variant="filled" href={activatePath(lic.key)}>
              <Icon.Add />
              Activate a machine
            </Button.Button>
          )}
        </>
      }
    >
      {rename.error != null && (
        <Status.Summary variant="error" level="small" message={rename.error} />
      )}
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
                <Text.Text level="p" overflow="fade">
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
                <Text.Text level="p" color={9} overflow="fade">
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
