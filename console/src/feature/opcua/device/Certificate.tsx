// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Certificate.css";

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
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Rack, Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { executeScanCommand } from "@/feature/opcua/device/scanCommand";
import {
  type CertificateInfo,
  type CertificatesResponse,
  certificatesResponseZ,
  GET_CERTIFICATE_COMMAND_TYPE,
  REGENERATE_CERTIFICATE_COMMAND_TYPE,
} from "@/feature/opcua/task/types";
import { Clipboard } from "@/platform/clipboard";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

const COMMAND_TIMEOUT = TimeSpan.seconds(10);

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
  new TimeStamp(notAfter).toString("ISODate");

const executeCertificateCommand = async (
  client: Client,
  rack: rack.Key,
  type: string,
): Promise<CertificatesResponse> =>
  await executeScanCommand(client, rack, {
    type,
    args: {},
    schema: certificatesResponseZ,
    timeout: COMMAND_TIMEOUT,
  });

interface DetailProps extends Description.ValueProps {
  label: string;
}

const Detail = ({ label, ...rest }: DetailProps): ReactElement => (
  <Description.Item>
    <Description.Label>{label}</Description.Label>
    <Description.Value {...rest} />
  </Description.Item>
);

interface CopyButtonProps {
  thumbprint: string;
  name: string;
}

const CopyButton = ({ thumbprint, name }: CopyButtonProps): ReactElement => {
  const copy = Clipboard.useCopy();
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={() => copy(thumbprint, "thumbprint", name)}
    >
      Copy
    </Button.Button>
  );
};

interface ThumbprintProps extends PropsWithChildren {
  value: string;
  name: string;
}

const Thumbprint = ({ value, name, children }: ThumbprintProps): ReactElement => (
  <Flex.Box x align="center" gap="medium" className={CSS.BE("opc-certificate", "line")}>
    <Text.Text level="small" color={9}>
      Thumbprint
    </Text.Text>
    <Text.Text
      level="small"
      variant="code"
      className={CSS.BE("opc-certificate", "value")}
    >
      {formatThumbprint(value)}
    </Text.Text>
    <Flex.Box x gap="tiny" align="center">
      <CopyButton thumbprint={value} name={name} />
      {children}
    </Flex.Box>
  </Flex.Box>
);

interface DriverCertificateDetailsProps {
  certificate: CertificateInfo;
}

const DriverCertificateDetails = ({
  certificate,
}: DriverCertificateDetailsProps): ReactElement => (
  <Description.List>
    <Detail label="Thumbprint">
      <Flex.Box x align="center" gap="small">
        <Text.Text variant="code">{formatThumbprint(certificate.thumbprint)}</Text.Text>
        <CopyButton thumbprint={certificate.thumbprint} name="Driver certificate" />
      </Flex.Box>
    </Detail>
    <Detail label="Expires">{formatExpiry(certificate.notAfter)}</Detail>
  </Description.List>
);

const Note = ({ children }: PropsWithChildren): ReactElement => (
  <Text.Text level="small" color={9}>
    {children}
  </Text.Text>
);

type CertificatesState =
  | { rack: rack.Key; variant: "success"; certificates: CertificatesResponse }
  | { rack: rack.Key; variant: "error"; status: status.Status };

const useCertificates = (rack: rack.Key) => {
  const client = Synnax.use();
  const [state, setState] = useState<CertificatesState | null>(null);
  const [loads, setLoads] = useState(0);
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
    [client, rack, loads],
  );
  return {
    state: state?.rack === rack ? state : null,
    setState,
    reload: () => setLoads((n) => n + 1),
  };
};

const useDownloadCertificate = () => {
  const download = Runtime.useDownload();
  const handleError = Status.useErrorHandler();
  return (certificate: CertificateInfo, name: string) =>
    handleError(
      async () =>
        await download({
          stream: Runtime.toStream(toDER(certificate.pem)),
          name,
          extension: "der",
        }),
      "Failed to download the Driver certificate",
    );
};

interface DriverCertificateModalParams {
  rackKey: rack.Key;
}

const DriverCertificateModal = ({
  rackKey,
}: Modals.ContentProps<DriverCertificateModalParams, void>): ReactElement => {
  const client = Synnax.use();
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  const downloadCertificate = useDownloadCertificate();
  Rack.useEnsure({ key: rackKey });
  const rackName = Rack.useName({ key: rackKey });
  const { state, setState } = useCertificates(rackKey);
  const certificates = state?.variant === "success" ? state.certificates : null;
  const current = certificates?.current;
  const renewal = certificates?.next;
  const handleRegenerate = () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const confirmed = await confirm({
        message: "Regenerate Driver certificate",
        description:
          "Each server must trust the new certificate before the Driver can connect to it again.",
        confirm: { label: "Regenerate" },
      });
      if (confirmed !== true) return;
      const next = await executeCertificateCommand(
        client,
        rackKey,
        REGENERATE_CERTIFICATE_COMMAND_TYPE,
      );
      setState({ rack: rackKey, variant: "success", certificates: next });
    }, "Failed to regenerate the Driver certificate");
  let body: ReactElement;
  if (state == null) body = <Note>Loading the Driver certificate</Note>;
  else if (state.variant === "error") body = <Status.Summary status={state.status} />;
  else
    body = (
      <>
        <Text.Text color={9}>
          The Driver presents this certificate to each OPC UA server that it connects to
          with security.
        </Text.Text>
        <DriverCertificateDetails certificate={state.certificates.current} />
        {renewal != null && (
          <Flex.Box y gap="medium">
            <Text.Text level="h5" weight={500} color={11}>
              Renewal
            </Text.Text>
            <Text.Text color={9}>
              The Driver uses this certificate when every server trusts it. Add it to
              each server that does not accept it automatically.
            </Text.Text>
            <DriverCertificateDetails certificate={renewal} />
            <Flex.Box x>
              <Button.Button
                variant="outlined"
                onClick={() => downloadCertificate(renewal, "driver-renewal")}
              >
                <Icon.Download />
                Download renewal
              </Button.Button>
            </Flex.Box>
          </Flex.Box>
        )}
      </>
    );

  return (
    <Modal.Frame className={CSS.B("opc-driver-certificate")}>
      <Modal.Header icon={<Icon.Logo.OPCUA />}>
        {[rackName ?? "Driver", "OPC UA Driver certificate"]}
      </Modal.Header>
      <Modal.Body gap="large">{body}</Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End gap="small">
          <Button.Button
            variant="outlined"
            disabled={current == null}
            onClick={handleRegenerate}
          >
            <Icon.Refresh />
            Regenerate
          </Button.Button>
          <Button.Button
            variant="filled"
            disabled={current == null}
            onClick={() => current != null && downloadCertificate(current, "driver")}
          >
            <Icon.Download />
            Download
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens the certificate that the Driver on a rack presents to OPC UA servers, with
 * actions to download it and to regenerate it.
 */
export const useDriverCertificateModal = Modals.createPrompt<
  void,
  DriverCertificateModalParams
>(DriverCertificateModal);

export interface DriverCertificateProps {
  rack: rack.Key;
}

/**
 * Shows the thumbprint of the certificate the Driver on the rack presents to servers,
 * with actions to copy it, download it, and open the Driver certificate.
 */
export const DriverCertificate = ({ rack }: DriverCertificateProps): ReactElement => {
  const { state, reload } = useCertificates(rack);
  const downloadCertificate = useDownloadCertificate();
  const openModal = useDriverCertificateModal();
  if (rack === 0)
    return <Note>Select where to connect from to show its certificate.</Note>;
  if (state == null) return <Note>Loading the Driver certificate</Note>;
  if (state.variant === "error") return <Status.Summary status={state.status} />;
  const { current } = state.certificates;
  const handleManage = () => {
    void openModal({ rackKey: rack }).then(reload);
  };
  return (
    <Thumbprint value={current.thumbprint} name="Driver certificate">
      <Button.Button
        size="small"
        variant="text"
        tooltip="Download as DER"
        onClick={() => downloadCertificate(current, "driver")}
      >
        Download
      </Button.Button>
      <Button.Button size="small" variant="text" onClick={handleManage}>
        Manage
      </Button.Button>
    </Thumbprint>
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
  const pinned = pem === "" ? null : readThumbprint(pem);
  return (
    <Flex.Box y gap={2}>
      {pinned == null && offer == null && (
        <Text.Text
          level="small"
          color={9}
          className={CSS.BE("opc-certificate", "line")}
        >
          No certificate yet. Connect to get one from the server.
        </Text.Text>
      )}
      {typeof pinned === "string" && (
        <Thumbprint value={pinned} name="Server certificate" />
      )}
      {pinned != null && typeof pinned !== "string" && (
        <Status.Summary status={pinned} />
      )}
      {offer != null && (
        <Flex.Box
          y
          gap="medium"
          bordered
          borderColor={6}
          rounded="small"
          className={CSS.B("opc-certificate-offer")}
        >
          <Text.Text
            level="small"
            color={9}
            className={CSS.BE("opc-certificate-offer", "note")}
          >
            The server sent this certificate. Compare its thumbprint with the
            certificate on the server, then trust it.
          </Text.Text>
          <Description.List level="small">
            <Detail label="Subject">{offer.subject}</Detail>
            <Detail label="Thumbprint" variant="code">
              {formatThumbprint(offer.thumbprint)}
            </Detail>
            <Detail label="Expires">{formatExpiry(offer.notAfter)}</Detail>
          </Description.List>
          <Flex.Box x>
            <Button.Button size="small" variant="filled" onClick={() => onTrust(offer)}>
              Trust
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
