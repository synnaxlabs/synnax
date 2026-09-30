// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/useInfoModal.css";

import { license, type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { caseconv, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

interface Standing {
  label: string;
  variant: status.Variant;
}

const standingOf = ({ state, warning }: license.Info): Standing => {
  if (state === "expired") return { label: "Expired", variant: "error" };
  if (state === "missing") return { label: "Inactive", variant: "warning" };
  if (warning !== "") return { label: "Expiring", variant: "warning" };
  return { label: "Active", variant: "success" };
};

const BINDINGS: Record<License.Binding, string> = {
  floating: "Runs on any machine",
  host: "Bound to this machine",
  other: "Bound to another machine",
};

interface StatProps {
  label: string;
  value: string;
  caption?: string;
  captionStatus?: status.Variant;
}

const Stat = ({ label, value, caption, captionStatus }: StatProps): ReactElement => (
  <Flex.Box y gap="tiny" className={CSS.BE("license-info", "stat")}>
    <Text.Text level="small" color={9}>
      {label}
    </Text.Text>
    <Text.Text level="h3" weight={500} overflow="nowrap">
      {value}
    </Text.Text>
    {caption != null && (
      <Text.Text level="small" color={9} status={captionStatus}>
        {caption}
      </Text.Text>
    )}
  </Flex.Box>
);

interface StatsProps {
  license: license.License;
  fingerprint: string[];
}

const Stats = ({ license: lic, fingerprint }: StatsProps): ReactElement => {
  const binding = License.resolveBinding(lic, fingerprint);
  return (
    <div className={CSS.BE("license-info", "stats")}>
      <Stat
        label="Machines"
        value={lic.machines.toLocaleString()}
        caption={BINDINGS[binding]}
        captionStatus={binding === "other" ? "warning" : undefined}
      />
      <Stat
        label="Channels"
        value={lic.channels === 0 ? "Unlimited" : lic.channels.toLocaleString()}
        caption={lic.channels === 0 ? undefined : "Per Core"}
      />
      <Stat
        label={lic.exp == null ? "Term" : "Expires"}
        value={
          lic.exp == null
            ? "Perpetual"
            : TimeStamp.seconds(lic.exp)
                .date()
                .toLocaleDateString(undefined, { dateStyle: "medium" })
        }
        caption={
          lic.maxVersion == null ? undefined : `Versions up to ${lic.maxVersion}`
        }
      />
    </div>
  );
};

interface HeadlineProps {
  info: license.Info;
}

const Headline = ({ info }: HeadlineProps): ReactElement => {
  const { label, variant } = standingOf(info);
  const lic = info.license;
  const description =
    info.warning !== ""
      ? caseconv.capitalize(info.warning)
      : lic == null
        ? license.STATE_MESSAGES[info.state]
        : undefined;
  return (
    <Flex.Box y gap="small">
      <Flex.Box x gap="medium" align="center">
        <Text.Text level="h3" weight={500}>
          {lic == null ? "No license" : `${License.editionLabel(lic)} license`}
        </Text.Text>
        <Tag.Tag icon={<Status.Indicator variant={variant} />} size="small">
          {label}
        </Tag.Tag>
      </Flex.Box>
      {description != null && (
        <Text.Text color={9} status={variant === "error" ? "error" : undefined}>
          {description}
        </Text.Text>
      )}
    </Flex.Box>
  );
};

/**
 * Opens a dialog with the active Core's license and the facts support asks for: the
 * license's claims, the host fingerprint, and the Core and Console versions.
 */
export const useInfoModal = Modals.create(() => {
  const { info, error } = License.useInfo();
  const { details } = Synnax.useConnectionStatus();
  const core = Session.Core.useSelectSelected();
  const consoleVersion = Session.Version.use();
  const fingerprint = info?.fingerprint ?? [];
  const lic = info?.license;
  const address = core == null ? null : `${core.host}:${core.port}`;
  const getDiagnostics = (): string =>
    JSON.stringify(
      {
        core: { address, version: details.nodeVersion },
        console: { version: consoleVersion },
        license: info ?? null,
      },
      null,
      2,
    );
  const coreLine =
    details.nodeVersion == null
      ? null
      : `Core ${details.nodeVersion}${address == null ? "" : ` at ${address}`}`;
  return (
    <Modal.Frame className={CSS.B("license-info")}>
      <Modal.Header icon={<Icon.License />}>License</Modal.Header>
      <Modal.Body className={CSS.BE("license-info", "body")} gap="huge">
        {error != null ? (
          <Status.Summary
            variant="error"
            level="h4"
            message="Failed to read the license"
            description={error.message}
          />
        ) : (
          info != null && (
            <Flex.Box y gap="large">
              <Headline info={info} />
              {lic != null && <Stats license={lic} fingerprint={fingerprint} />}
            </Flex.Box>
          )
        )}
        <Flex.Box y gap="small" className={CSS.BE("license-info", "support")}>
          <Flex.Box y gap="tiny">
            {lic != null && (
              <>
                <Text.Text level="small" variant="code" color={9}>
                  License {lic.jti}
                </Text.Text>
                <Text.Text level="small" variant="code" color={9}>
                  Organization {lic.organization}
                </Text.Text>
              </>
            )}
            <Text.Text level="small" variant="code" color={9}>
              {coreLine == null
                ? `Console ${consoleVersion}`
                : `${coreLine}, Console ${consoleVersion}`}
            </Text.Text>
          </Flex.Box>
          <Flex.Box x gap="small" className={CSS.BE("license-info", "copy")}>
            <Button.Copy variant="text" size="small" text={getDiagnostics}>
              Copy diagnostics
            </Button.Copy>
            <Button.Copy
              variant="text"
              size="small"
              disabled={fingerprint.length === 0}
              text={() => License.joinFingerprint(fingerprint)}
            >
              Copy fingerprint
            </Button.Copy>
          </Flex.Box>
        </Flex.Box>
      </Modal.Body>
    </Modal.Frame>
  );
});
