// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/certificate/Details.css";

import { type certificate, type rack } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Description } from "@synnaxlabs/lyra/description";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { leafDER } from "@/platform/certificate/pem";
import { CSS } from "@/platform/css";
import { Picker } from "@/platform/picker";
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
      <Dialog.Trigger
        hideCaret
        size="small"
        variant="text"
        textColor={9}
        tooltip="Download"
      >
        <Icon.Download />
      </Dialog.Trigger>
      <Menu.Dialog>
        <DownloadMenu onDownload={handleDownload} />
      </Menu.Dialog>
    </Dialog.Frame>
  );
};

interface FingerprintProps {
  certificate: certificate.Certificate;
  sha256: string;
}

/** The fingerprint on one line, with its copy and download buttons at the end. */
const Fingerprint = ({ certificate: c, sha256 }: FingerprintProps): ReactElement => {
  const text = formatFingerprint(sha256);
  return (
    <Description.Item>
      <Description.Label>SHA-256</Description.Label>
      <Description.Value align="center">
        <Text.Text
          el="span"
          level="small"
          variant="code"
          className={CSS.BE("certificate-field", "fingerprint")}
        >
          {text}
        </Text.Text>
        <Button.Copy
          text={text}
          size="small"
          variant="text"
          textColor={9}
          tooltip="Copy fingerprint"
        />
        {c.source === "stored" && <Download certificate={c} />}
      </Description.Value>
    </Description.Item>
  );
};

interface CardProps {
  certificate: certificate.Certificate;
  now: TimeStamp;
}

const Card = ({ certificate: c, now }: CardProps): ReactElement => {
  const { details } = c;
  const validity = Secret.Validity.describeCertificate(c, now);
  return (
    <Flex.Box
      y
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
            <Fingerprint certificate={c} sha256={details.sha256} />
          </>
        )}
      </Description.List>
    </Flex.Box>
  );
};

export interface DetailsProps {
  certificate: certificate.Certificate;
  /** The time the validity reads against. */
  now: TimeStamp;
  /** The rack of the device. A file certificate of another rack shows an error. */
  rack: rack.Key;
}

/**
 * A line with the path of a file certificate and a button that shows the full details
 * of the certificate. A stored certificate also downloads.
 */
export const Details = ({ certificate: c, now, rack }: DetailsProps): ReactElement => {
  const [visible, setVisible] = useState(false);
  return (
    <Flex.Box y gap="small">
      <Picker.Line justify="end">
        {c.source === "file" && (
          <Text.Text
            level="small"
            color={9}
            overflow="fade"
            className={CSS.BE("certificate-field", "summary-text")}
          >
            {c.rack !== rack && (
              <>
                <Text.Text
                  el="span"
                  color={9}
                  status={Secret.OTHER_RACK_DESCRIPTION.status}
                >
                  {Secret.OTHER_RACK_DESCRIPTION.text}
                </Text.Text>
                <span>·</span>
              </>
            )}
            <span>{c.certificatePath}</span>
          </Text.Text>
        )}
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => setVisible(!visible)}
        >
          {visible ? "Hide details" : "Details"}
        </Button.Button>
      </Picker.Line>
      {visible && <Card certificate={c} now={now} />}
    </Flex.Box>
  );
};
