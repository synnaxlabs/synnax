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
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Machine } from "@/server/license/desktop";
import { LogOutDialog } from "@/ui/devices/LogOutDialog";
import { Enterprise } from "@/ui/Enterprise";
import { date, machineName, statusOf } from "@/ui/format";
import { StatusTag } from "@/ui/licenses/StatusTag";
import { DOWNLOAD_URL } from "@/ui/links";
import { Page, Section } from "@/ui/Page";
import { Panel } from "@/ui/Panel";
import { Row, Table } from "@/ui/Table";

/** RECENT is how many machines the overview shows before linking to Devices. */
const RECENT = 3;

const COLUMNS = "minmax(0, 2fr) 14rem 16rem 10rem";

export interface PersonalProps {
  /** machines are the user's linked machines, most recently renewed first. */
  machines: Machine[];
  /** devices is the URL of the Devices tab. */
  devices: string;
  now: Date | string;
}

/**
 * Personal is the overview of a personal account: how to start with Synnax Desktop,
 * or the machines already logged in.
 */
export const Personal = ({ machines, devices, now }: PersonalProps): ReactElement => (
  <Page title="Synnax Desktop" subtitle="Free for personal use">
    {machines.length === 0 ? (
      <GetStarted />
    ) : (
      <Recent machines={machines} devices={devices} now={new Date(now)} />
    )}
    <Enterprise />
  </Page>
);

interface StepProps {
  step: number;
  title: string;
  description: string;
  action?: ReactElement;
}

const Step = ({ step, title, description, action }: StepProps): ReactElement => (
  <Flex.Box x align="center" gap="large" wrap className="portal-step">
    <Text.Text level="h5" color={10} className="portal-step__number">
      {step}
    </Text.Text>
    <Flex.Box y gap="tiny" style={{ flex: "1 1 30rem" }}>
      <Text.Text level="h5" weight={500} color={11}>
        {title}
      </Text.Text>
      <Text.Text level="p" color={9}>
        {description}
      </Text.Text>
    </Flex.Box>
    {action}
  </Flex.Box>
);

const GetStarted = (): ReactElement => (
  <Section title="Get started">
    <Panel>
      <Step
        step={1}
        title="Download Synnax Desktop"
        description="Available for macOS and Windows"
        action={
          <Button.Button variant="filled" href={DOWNLOAD_URL}>
            <Icon.Download />
            Download
          </Button.Button>
        }
      />
      <Step
        step={2}
        title="Log in from the app"
        description="Open Synnax Desktop and choose Log in. The machine then shows here"
      />
    </Panel>
  </Section>
);

interface RecentProps {
  machines: Machine[];
  devices: string;
  now: Date;
}

const Recent = ({ machines, devices, now }: RecentProps): ReactElement => (
  <Section
    title="Recent devices"
    actions={
      <Button.Button variant="text" textColor={9} href={devices}>
        View all
      </Button.Button>
    }
  >
    <Table columns={COLUMNS} head={["Machine", "Status", "Last renewal", ""]}>
      {machines.slice(0, RECENT).map(({ activation: a, license: lic }) => (
        <Row key={a.key} columns={COLUMNS}>
          <Text.Text level="p" overflow="ellipsis">
            {machineName(a)}
          </Text.Text>
          <Flex.Box>
            <StatusTag status={statusOf(lic, now)} />
          </Flex.Box>
          <Text.Text level="p" color={9}>
            {date(a.lastSeen)}
          </Text.Text>
          <Flex.Box justify="end">
            <LogOutDialog activation={a} />
          </Flex.Box>
        </Row>
      ))}
    </Table>
    <Button.Button variant="text" size="small" textColor={9} href={DOWNLOAD_URL}>
      <Icon.Download />
      Download Synnax Desktop for another machine
    </Button.Button>
  </Section>
);
