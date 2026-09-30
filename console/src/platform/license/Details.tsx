// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { license } from "@synnaxlabs/client";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { describeChannels, describeTerm, editionLabel } from "@/platform/license/term";
import { useInfo } from "@/platform/license/useInfo";

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
    <Flex.Box y gap="small" className={CSS.B("license")}>
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
 * The license in full for the version info modal. Renders nothing when the user may not
 * read the license.
 */
export const Details = (): ReactElement | null =>
  Access.useRetrieveGranted(license.ONTOLOGY_ID) ? <GrantedDetails /> : null;
