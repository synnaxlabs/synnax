// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sha1 } from "@noble/hashes/legacy.js";
import { bytesToHex } from "@noble/hashes/utils.js";
import {
  DisconnectedError,
  type rack,
  status,
  type Synnax as Client,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { retrieveScanTask } from "@/feature/opcua/device/retrieveScanTask";
import {
  type CertificateInfo,
  type CertificatesResponse,
  certificatesResponseZ,
  GET_CERTIFICATE_COMMAND_TYPE,
  REGENERATE_CERTIFICATE_COMMAND_TYPE,
} from "@/feature/opcua/device/scan";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

const COMMAND_TIMEOUT = TimeSpan.seconds(10);

const toStream = (bytes: Uint8Array): ReadableStream<Uint8Array> =>
  new ReadableStream({
    start: (controller) => {
      controller.enqueue(bytes);
      controller.close();
    },
  });

const PEM_ARMOR = /-----(BEGIN|END) CERTIFICATE-----|\s/g;

const toDER = (pem: string): Uint8Array<ArrayBuffer> =>
  Uint8Array.from(atob(pem.replace(PEM_ARMOR, "")), (c) => c.charCodeAt(0));

const readThumbprint = (pem: string): string | status.Status => {
  try {
    return bytesToHex(sha1(toDER(pem))).toUpperCase();
  } catch (error) {
    return status.fromException(error, "Failed to read the server certificate");
  }
};

const formatThumbprint = (thumbprint: string): string =>
  thumbprint.match(/.{1,2}/g)?.join(":") ?? thumbprint;

const formatExpiry = (notAfter: string): string =>
  `Expires ${new TimeStamp(notAfter).toString("ISODate")}`;

const executeCertificateCommand = async (
  client: Client,
  rack: rack.Key,
  type: string,
): Promise<CertificatesResponse> => {
  const scanTask = await retrieveScanTask(client, rack);
  const res = await scanTask.executeCommandSync({
    type,
    timeout: COMMAND_TIMEOUT,
    args: {},
  });
  if (res.variant === "error") throw status.toError(res);
  return certificatesResponseZ.parse(res.details?.data);
};

interface DetailsProps extends PropsWithChildren {
  title?: string;
  subject?: string;
  thumbprint: string;
  notAfter?: string;
}

const Details = ({
  title,
  subject,
  thumbprint,
  notAfter,
  children,
}: DetailsProps): ReactElement => (
  <Flex.Box x align="center" justify="between" gap="medium">
    <Flex.Box y gap="tiny">
      {title != null && (
        <Text.Text level="small" weight={500}>
          {title}
        </Text.Text>
      )}
      {subject != null && <Text.Text level="small">{subject}</Text.Text>}
      <Text.Text level="small" variant="code">
        {formatThumbprint(thumbprint)}
      </Text.Text>
      {notAfter != null && (
        <Text.Text level="small" color={9}>
          {formatExpiry(notAfter)}
        </Text.Text>
      )}
    </Flex.Box>
    {children != null && (
      <Flex.Box x gap="small">
        {children}
      </Flex.Box>
    )}
  </Flex.Box>
);

type CertificatesState =
  | { rack: rack.Key; variant: "success"; certificates: CertificatesResponse }
  | { rack: rack.Key; variant: "error"; status: status.Status };

export interface DriverCertificateProps {
  rack: rack.Key;
}

/**
 * Shows the certificate the Driver on the rack presents to servers, with actions to
 * download it and to regenerate it.
 */
export const DriverCertificate = ({ rack }: DriverCertificateProps): ReactElement => {
  const client = Synnax.use();
  const confirm = Modals.useConfirm();
  const download = Runtime.useDownload();
  const handleError = Status.useErrorHandler();
  const [state, setState] = useState<CertificatesState | null>(null);
  useAsyncEffect(
    async (signal) => {
      if (client == null || rack === 0) return;
      let next: CertificatesState;
      try {
        const certificates = await executeCertificateCommand(
          client,
          rack,
          GET_CERTIFICATE_COMMAND_TYPE,
        );
        next = { rack, variant: "success", certificates };
      } catch (error) {
        next = {
          rack,
          variant: "error",
          status: status.fromException(error, "Failed to load the Driver certificate"),
        };
      }
      if (!signal.aborted) setState(next);
    },
    [client, rack],
  );
  if (rack === 0)
    return (
      <Input.HelpText>Select a rack to show its Driver certificate.</Input.HelpText>
    );
  if (state?.rack !== rack)
    return <Input.HelpText>Loading the Driver certificate</Input.HelpText>;
  if (state.variant === "error") return <Status.Summary status={state.status} />;
  const { current, next } = state.certificates;
  const handleDownload = (certificate: CertificateInfo, name: string) =>
    handleError(
      async () =>
        await download({
          stream: toStream(toDER(certificate.pem)),
          name,
          extension: "der",
        }),
      "Failed to download the Driver certificate",
    );
  const handleRegenerate = () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const confirmed = await confirm({
        message: "Are you sure you want to regenerate the Driver certificate?",
        description:
          "Every server must trust the new certificate before the Driver can connect to it again.",
        confirm: { label: "Regenerate" },
      });
      if (confirmed !== true) return;
      const certificates = await executeCertificateCommand(
        client,
        rack,
        REGENERATE_CERTIFICATE_COMMAND_TYPE,
      );
      setState({ rack, variant: "success", certificates });
    }, "Failed to regenerate the Driver certificate");
  return (
    <Flex.Box y gap="small">
      <Details thumbprint={current.thumbprint} notAfter={current.notAfter}>
        <Button.Button
          size="small"
          variant="text"
          tooltip="Download as DER"
          onClick={() => handleDownload(current, "driver")}
        >
          <Icon.Download />
          Download
        </Button.Button>
        <Button.Button size="small" variant="text" onClick={handleRegenerate}>
          <Icon.Refresh />
          Regenerate
        </Button.Button>
      </Details>
      {next != null && (
        <Details
          title="Next certificate"
          thumbprint={next.thumbprint}
          notAfter={next.notAfter}
        >
          <Button.Button
            size="small"
            variant="text"
            tooltip="Download as DER"
            onClick={() => handleDownload(next, "driver-next")}
          >
            <Icon.Download />
            Download
          </Button.Button>
        </Details>
      )}
    </Flex.Box>
  );
};

export interface ServerCertificateProps {
  /** The pinned certificate as PEM, or empty when the device trusts none. */
  pem: string;
  /** The certificate the server presented on a failed connect. */
  offer: CertificateInfo | null;
  onTrust: (offer: CertificateInfo) => void;
}

/** Shows the server certificate the device trusts, and offers one the server sent. */
export const ServerCertificate = ({
  pem,
  offer,
  onTrust,
}: ServerCertificateProps): ReactElement => {
  let pinned: ReactElement;
  if (pem === "") pinned = <Text.Text level="small">Not trusted</Text.Text>;
  else {
    const value = readThumbprint(pem);
    pinned =
      typeof value === "string" ? (
        <Details thumbprint={value} />
      ) : (
        <Status.Summary status={value} />
      );
  }
  return (
    <Flex.Box y gap="small">
      {pinned}
      {offer != null && (
        <Details
          title="The server sent this certificate"
          subject={offer.subject}
          thumbprint={offer.thumbprint}
          notAfter={offer.notAfter}
        >
          <Button.Button
            size="small"
            variant="filled"
            tooltip="Compare the thumbprint with the server before you trust it"
            onClick={() => onTrust(offer)}
          >
            Trust
          </Button.Button>
        </Details>
      )}
    </Flex.Box>
  );
};
