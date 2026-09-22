// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/license/Details.css";

import { type license } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { describeChannels, describeTerm, editionLabel } from "@/platform/license/term";
import { ACCOUNT_URL } from "@/platform/license/urls";
import { useInfo } from "@/platform/license/useInfo";
import { Session } from "@/session";

const STATE_MESSAGES: Record<license.State, string> = {
  ok: "Licensed",
  missing: "No license is active on this Core",
  expired: "The license on this Core has expired",
};

/** The license on two lines for the Core badge: what applies, and any warning. */
export const Summary = (): ReactElement | null => {
  const { info } = useInfo();
  if (info == null) return null;
  const { state, warning, license } = info;
  const label =
    license == null || state !== "ok"
      ? STATE_MESSAGES[state]
      : `${editionLabel(license)} license, ${describeTerm(license).toLowerCase()}`;
  return (
    <>
      <Text.Text
        level="small"
        status={state === "ok" ? undefined : "warning"}
        color={9}
      >
        {label}
      </Text.Text>
      {warning != null && (
        <Text.Text level="small" status="warning">
          {warning}
        </Text.Text>
      )}
    </>
  );
};

interface RowProps {
  name: string;
  value: string;
}

const Row = ({ name, value }: RowProps): ReactElement => (
  <Flex.Box x justify="between" gap="large" className={CSS.BE("license", "row")}>
    <Text.Text level="small" color={9}>
      {name}
    </Text.Text>
    <Text.Text level="small" color={10} overflow="ellipsis">
      {value}
    </Text.Text>
  </Flex.Box>
);

/** The license in full for the version info modal. */
export const Details = (): ReactElement | null => {
  const { info, error } = useInfo();
  let body: ReactElement | null = null;
  if (error != null)
    body = (
      <Status.Summary
        variant="error"
        level="small"
        message="Failed to read the license"
        description={error.message}
      />
    );
  else if (info != null) {
    const { state, warning, license } = info;
    body = (
      <>
        {license == null || state !== "ok" ? (
          <Status.Summary
            variant="warning"
            level="small"
            message={STATE_MESSAGES[state]}
          />
        ) : (
          <>
            <Row name="Edition" value={editionLabel(license)} />
            <Row name="Organization" value={license.org} />
            <Row name="Term" value={describeTerm(license)} />
            <Row name="Hosts" value={String(license.n)} />
            <Row name="Channels" value={describeChannels(license)} />
          </>
        )}
        {warning != null && (
          <Status.Summary variant="warning" level="small" message={warning} />
        )}
      </>
    );
  }
  return (
    <Flex.Box y gap="small" className={CSS.B("license")}>
      <Account />
      {body}
    </Flex.Box>
  );
};

/** The account a Synnax Desktop machine is linked to, with a way to the hub. */
const Account = (): ReactElement | null => {
  const { email } = Session.Account.useSelect();
  if (email == null) return null;
  return (
    <Flex.Box
      x
      justify="between"
      align="center"
      gap="large"
      className={CSS.BE("license", "row")}
    >
      <Text.Text level="small" color={10} overflow="ellipsis">
        Signed in as {email}
      </Text.Text>
      <Button.Button variant="text" size="small" href={ACCOUNT_URL} target="_blank">
        <Icon.OpenExternal />
        Manage in your account
      </Button.Button>
    </Flex.Box>
  );
};
