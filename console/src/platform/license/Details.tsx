// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { license } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { describeChannels, describeTerm, editionLabel } from "@/platform/license/term";
import { ACCOUNT_URL } from "@/platform/license/urls";
import { useInfo } from "@/platform/license/useInfo";
import { Session } from "@/session";

const GrantedSummary = (): ReactElement | null => {
  const { info } = useInfo();
  if (info == null) return null;
  const { state, warning, license: lic } = info;
  const label =
    lic == null || state !== "ok"
      ? license.STATE_MESSAGES[state]
      : `${editionLabel(lic)} license, ${describeTerm(lic).toLowerCase()}`;
  return (
    <>
      <Text.Text
        level="small"
        status={state === "ok" ? undefined : "warning"}
        color={9}
      >
        {label}
      </Text.Text>
      {warning !== "" && (
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
  <Description.Item>
    <Description.Label>{name}</Description.Label>
    <Description.Value overflow="ellipsis">{value}</Description.Value>
  </Description.Item>
);

const GrantedDetails = (): ReactElement | null => {
  const { info, error } = useInfo();
  if (error != null)
    return (
      <Status.Summary
        variant="error"
        level="small"
        message="Failed to read the license"
        description={error.message}
      />
    );
  if (info == null) return null;
  const { state, warning, license: lic } = info;
  return (
    <>
      {lic == null || state !== "ok" ? (
        <Status.Summary
          variant="warning"
          level="small"
          message={license.STATE_MESSAGES[state]}
        />
      ) : (
        <Description.List level="small" justify="between">
          <Row name="Edition" value={editionLabel(lic)} />
          <Row name="Organization" value={lic.organization} />
          <Row name="Term" value={describeTerm(lic)} />
          <Row name="Machines" value={String(lic.machines)} />
          <Row name="Channels" value={describeChannels(lic)} />
        </Description.List>
      )}
      {warning !== "" && (
        <Status.Summary variant="warning" level="small" message={warning} />
      )}
    </>
  );
};

/** The account a Synnax Desktop machine is linked to, with a way to the hub. */
const Account = (): ReactElement | null => {
  const email = Session.Account.useSelectEmail();
  if (email == null) return null;
  return (
    <Flex.Box x justify="between" align="center" gap="large">
      <Text.Text level="small" color={10} overflow="ellipsis">
        Logged in as {email}
      </Text.Text>
      <Button.Button variant="text" size="small" href={ACCOUNT_URL} target="_blank">
        <Icon.OpenExternal />
        Manage in your account
      </Button.Button>
    </Flex.Box>
  );
};

/**
 * The license on two lines for the Core badge: what applies, and any warning. Renders
 * nothing when the user may not read the license.
 */
export const Summary = (): ReactElement | null =>
  Access.useRetrieveGranted(license.ONTOLOGY_ID) ? <GrantedSummary /> : null;

/**
 * The account the machine is linked to, and the license in full, for the version info
 * modal. The license shows only when the user may read it.
 */
export const Details = (): ReactElement => {
  const granted = Access.useRetrieveGranted(license.ONTOLOGY_ID);
  return (
    <Flex.Box y gap="small" className={CSS.B("license")}>
      <Account />
      {granted && <GrantedDetails />}
    </Flex.Box>
  );
};
