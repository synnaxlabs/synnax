// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/certificate/Details.css";

import { type certificate } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Description } from "@synnaxlabs/lyra/description";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { leafDER } from "@/platform/certificate/pem";
import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";
import { Secret } from "@/platform/secret";

const formatFingerprint = (hex: string): string =>
  hex.toUpperCase().match(/.{2}/g)?.join(":") ?? hex;

const encoder = new TextEncoder();

type Format = "pem" | "der";

interface DetailProps extends Description.ValueProps {
  label: string;
}

const Detail = ({ label, ...rest }: DetailProps): ReactElement => (
  <Description.Item>
    <Description.Label>{label}</Description.Label>
    <Description.Value {...rest} />
  </Description.Item>
);

interface DownloadMenuProps {
  onDownload: (format: Format) => void;
}

const DownloadMenu = ({ onDownload }: DownloadMenuProps): ReactElement => {
  const { close } = Dialog.useContext();
  const select = (format: Format) => (): void => {
    close();
    onDownload(format);
  };
  return (
    <Menu.Menu
      level="small"
      gap="small"
      onChange={{ pem: select("pem"), der: select("der") }}
    >
      <Menu.Item itemKey="pem">Download as PEM</Menu.Item>
      <Menu.Item itemKey="der">Download as DER</Menu.Item>
    </Menu.Menu>
  );
};

interface DownloadProps {
  certificate: certificate.StoredCertificate;
}

/** Saves the chain as PEM, or the leaf as DER. */
const Download = ({ certificate: c }: DownloadProps): ReactElement => {
  const download = Runtime.useDownload();
  const handleError = Status.useErrorHandler();
  const handleDownload = (format: Format): void =>
    handleError(
      async () =>
        await download({
          stream: Runtime.toStream(
            format === "pem" ? encoder.encode(c.pem) : leafDER(c.pem),
          ),
          name: c.name,
          extension: format,
        }),
      "Failed to download certificate",
    );
  return (
    <Dialog.Frame>
      <Dialog.Trigger hideCaret size="small" variant="text" textColor={9}>
        <Icon.Download />
        Download
      </Dialog.Trigger>
      <Menu.Dialog>
        <DownloadMenu onDownload={handleDownload} />
      </Menu.Dialog>
    </Dialog.Frame>
  );
};

export interface DetailsProps {
  certificate: certificate.Certificate;
  /** The time the validity reads against. */
  now: TimeStamp;
}

/**
 * The subject, issuer, validity, and SHA-256 fingerprint of a certificate. A stored
 * certificate also downloads.
 */
export const Details = ({ certificate: c, now }: DetailsProps): ReactElement => {
  const { details } = c;
  const validity = Secret.Validity.describeCertificate(c, now);
  return (
    <Flex.Box
      y
      gap="small"
      bordered
      borderColor={6}
      rounded="small"
      background={1}
      className={CSS.BE("certificate-field", "details")}
    >
      <Description.List level="small">
        {c.source === "file" && <Detail label="Path">{c.certificatePath}</Detail>}
        {details != null && (
          <>
            <Detail label="Subject">{details.subject}</Detail>
            <Detail label="Issuer">{details.issuer}</Detail>
            <Detail label="Validity" status={validity.status}>
              {validity.text}
            </Detail>
            <Detail label="SHA-256" variant="code">
              {formatFingerprint(details.sha256)}
            </Detail>
          </>
        )}
      </Description.List>
      {details != null && (
        <Flex.Box x gap="small" align="center" justify="end">
          <Button.Copy
            text={formatFingerprint(details.sha256)}
            size="small"
            variant="text"
            textColor={9}
          >
            Copy fingerprint
          </Button.Copy>
          {c.source === "stored" && <Download certificate={c} />}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
