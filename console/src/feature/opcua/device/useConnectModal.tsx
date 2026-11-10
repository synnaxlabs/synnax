// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Connect.css";

import { type device, status, TimeSpan } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useState } from "react";

import {
  DriverCertificate,
  ServerCertificate,
} from "@/feature/opcua/device/Certificate";
import { commandTimeout } from "@/feature/opcua/device/commandTimeout";
import { Endpoints } from "@/feature/opcua/device/Endpoints";
import { sendScanCommand } from "@/feature/opcua/device/scanCommand";
import {
  SelectSecurityMode,
  type SelectSecurityModeProps,
} from "@/feature/opcua/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/feature/opcua/device/SelectSecurityPolicy";
import {
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  type Device,
  INTEGRATION_KEY,
  NO_SECURITY_MODE,
  NO_SECURITY_POLICY,
  SCHEMAS,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_PROPERTIES,
} from "@/feature/opcua/device/types";
import {
  type CertificateInfo,
  TEST_CONNECTION_COMMAND_TYPE,
  testConnectionResponseZ,
} from "@/feature/opcua/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";

const path = deep.path<Device>();

const INITIAL_VALUES: Device = {
  key: "",
  name: "OPC UA server",
  make: "opc",
  model: "OPC UA server",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => set(path.location, get(path.properties.connection.endpoint).value);

type ClientCertificateSource = "driver" | "files";

const DEPRECATED_SECURITY_POLICIES = new Set<SecurityPolicy>([
  BASIC256_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
]);

export const Connect = ({
  deviceKey,
  rack: defaultRack,
  close,
  onBack,
}: PlatformDevice.ConnectProps) => {
  const { capture } = Analytics.use();
  const [serverCertificateOffer, setServerCertificateOffer] =
    useState<CertificateInfo | null>(null);
  const [clientCertificateRejection, setClientCertificateRejection] = useState<
    string | null
  >(null);
  const [clientCertificateSource, setClientCertificateSource] =
    useState<ClientCertificateSource | null>(null);

  const beforeSave = async ({
    client,
    get,
    set,
  }: Flux.FormBeforeSaveParams<
    PDevice.RetrieveQuery,
    PDevice.FormSchema<typeof SCHEMAS>
  >) => {
    setServerCertificateOffer(null);
    setClientCertificateRejection(null);
    const connection = get(path.properties.connection).value;
    const scanStatus = await sendScanCommand(client, get(path.rack).value, {
      type: TEST_CONNECTION_COMMAND_TYPE,
      timeout: commandTimeout(connection),
      args: { connection },
    });
    if (scanStatus.variant === "error") {
      const reply = testConnectionResponseZ.safeParse(scanStatus.details?.data);
      if (reply.success && reply.data.serverCertificate != null)
        setServerCertificateOffer(reply.data.serverCertificate);
      else if (reply.success && reply.data.clientCertificateRejected === true)
        setClientCertificateRejection(scanStatus.description);
      else throw status.toError(scanStatus);
      // The certificate fields show the details, so the footer keeps only the message.
      throw status.toError({ ...scanStatus, description: "" });
    }
    // Since we just scanned successfully, we create a default healthy status for the
    // device that can then be overwritten by the scanner if we lose connection.
    const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
      name: get(path.name).value,
      message: "Server connected",
      variant: "success",
      details: {
        rack: get(path.rack).value,
        connected: true,
      },
    });
    set(path.status, devStatus, { notifyOnChange: false, markTouched: false });
    return true;
  };

  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: deviceKey == null ? null : { key: deviceKey },
    initialValues: { ...INITIAL_VALUES, rack: defaultRack ?? 0 },
    beforeValidate,
    beforeSave,
    afterSave: ({ get }) => {
      if (deviceKey == null) capture("device_connected", { integration: "opcua" });
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "OPC UA",
    rack: rackKey,
  });

  const { get: getField, set: setField } = form;
  const hasSecurity =
    Form.useFieldValue(path.properties.connection.securityMode, { ctx: form }) !=
    NO_SECURITY_MODE;
  const clientCertificate = Form.useFieldValue(
    path.properties.connection.clientCertificate,
    { ctx: form },
  );
  const clientPrivateKey = Form.useFieldValue(
    path.properties.connection.clientPrivateKey,
    { ctx: form },
  );
  const serverCertificate =
    Form.useFieldValue(path.properties.connection.serverCertificate, { ctx: form }) ??
    "";
  // Empty paths select the Driver certificate, so a device that names files has chosen
  // them until the user picks again.
  const hasFiles = (clientCertificate ?? "") !== "" || (clientPrivateKey ?? "") !== "";
  const source = clientCertificateSource ?? (hasFiles ? "files" : "driver");

  const handleSourceChange = (next: ClientCertificateSource) => {
    setClientCertificateSource(next);
    setClientCertificateRejection(null);
    if (next !== "driver") return;
    setField(path.properties.connection.clientCertificate, "");
    setField(path.properties.connection.clientPrivateKey, "");
  };

  const handleEndpointSelect = (mode: SecurityMode, policy: SecurityPolicy) => {
    setField(path.properties.connection.securityMode, mode);
    setField(path.properties.connection.securityPolicy, policy);
  };

  const handleTrust = (offer: CertificateInfo) => {
    setField(path.properties.connection.serverCertificate, offer.pem);
    save();
  };

  const renderSecurityMode = ({
    onChange,
    ...rest
  }: SelectSecurityModeProps & { onChange: (v: SecurityMode) => void }) => {
    const handleChange = (mode: SecurityMode) => {
      onChange(mode);
      const policyPath = path.properties.connection.securityPolicy;
      // Security mode None pairs only with policy None, and every other mode with a
      // policy other than None.
      if (mode === NO_SECURITY_MODE) setField(policyPath, NO_SECURITY_POLICY);
      else if (getField(policyPath).value === NO_SECURITY_POLICY)
        setField(policyPath, BASIC256_SHA256_SECURITY_POLICY);
    };
    return (
      <Flex.Box x justify="between" align="start" grow>
        <SelectSecurityMode {...rest} onChange={handleChange} />
        <Endpoints rack={rackKey} onSelect={handleEndpointSelect} />
      </Flex.Box>
    );
  };

  return (
    <PlatformDevice.ConnectFrame>
      <Modal.Header icon={<Icon.Logo.OPCUA />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections>
            <Form.Section title="Connection">
              <Form.TextField
                path={path.properties.connection.endpoint}
                label="Endpoint"
                padHelpText={false}
                inputProps={ENDPOINT_INPUT_PROPS}
              />
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
              <Form.Field
                path={path.properties.connection.username}
                label="Username"
                padHelpText={false}
              >
                {renderLogin}
              </Form.Field>
            </Form.Section>
            <Form.Section title="Security">
              <Form.Field
                path={path.properties.connection.securityMode}
                label="Mode"
                padHelpText={false}
              >
                {renderSecurityMode}
              </Form.Field>
              {hasSecurity && (
                <Form.Field
                  path={path.properties.connection.securityPolicy}
                  label="Policy"
                  padHelpText={false}
                >
                  {renderSecurityPolicy}
                </Form.Field>
              )}
              {hasSecurity && (
                <>
                  <Input.Item
                    label="Client certificate"
                    status={clientCertificateRejection == null ? undefined : "error"}
                    helpText={clientCertificateRejection ?? undefined}
                  >
                    <Flex.Box y gap={2}>
                      <Select.Buttons<ClientCertificateSource>
                        value={source}
                        onChange={handleSourceChange}
                      >
                        <Select.Item<ClientCertificateSource> itemKey="driver">
                          Driver certificate
                        </Select.Item>
                        <Select.Item<ClientCertificateSource> itemKey="files">
                          Files on the Driver computer
                        </Select.Item>
                      </Select.Buttons>
                      {source === "driver" && <DriverCertificate rack={rackKey} />}
                    </Flex.Box>
                  </Input.Item>
                  {source === "files" && (
                    <>
                      <Form.TextField
                        path={path.properties.connection.clientCertificate}
                        label="Certificate file"
                        padHelpText={false}
                        inputProps={PATH_INPUT_PROPS}
                      />
                      <Form.TextField
                        path={path.properties.connection.clientPrivateKey}
                        label="Private key file"
                        padHelpText={false}
                        inputProps={PATH_INPUT_PROPS}
                      />
                    </>
                  )}
                  <Input.Item label="Server certificate">
                    <ServerCertificate
                      pem={serverCertificate}
                      offer={serverCertificateOffer}
                      onTrust={handleTrust}
                    />
                  </Input.Item>
                  <ValidationField />
                </>
              )}
            </Form.Section>
            <Form.Section title="Advanced">
              <Flex.Box x>
                <Form.TimeSpanField
                  path={path.properties.connection.clientTimeoutMs}
                  label="Request timeout"
                  unit={TimeSpan.MILLISECOND}
                  bounds={TIMING_BOUNDS}
                  padHelpText={false}
                />
                <Form.TimeSpanField
                  path={path.properties.connection.sessionTimeoutMs}
                  label="Session timeout"
                  unit={TimeSpan.MILLISECOND}
                  bounds={TIMING_BOUNDS}
                  padHelpText={false}
                />
                <Form.TimeSpanField
                  path={path.properties.connection.secureChannelLifetimeMs}
                  label="Channel lifetime"
                  unit={TimeSpan.MILLISECOND}
                  bounds={TIMING_BOUNDS}
                  padHelpText={false}
                />
              </Flex.Box>
            </Form.Section>
          </Form.Sections>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onBack={onBack}
      />
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const SecurityPolicyField = ({ value, onChange }: Input.Control<SecurityPolicy>) => (
  <>
    <SelectSecurityPolicy value={value} onChange={onChange} />
    {DEPRECATED_SECURITY_POLICIES.has(value) && (
      <Input.HelpText variant="warning">
        Use only if the server supports nothing newer.
      </Input.HelpText>
    )}
  </>
);

const renderSecurityPolicy = Component.renderProp(SecurityPolicyField);

/** The username, with the password beside it at the same width. */
const LoginField = ({ value, onChange }: Input.Control<string | undefined>) => (
  <div className={CSS.BE("opc-connect", "login")}>
    <Input.Text value={value ?? ""} onChange={onChange} placeholder="Anonymous" />
    <Form.TextField
      path={path.properties.connection.password}
      label="Password"
      padHelpText={false}
      inputProps={PASSWORD_INPUT_PROPS}
    />
  </div>
);

const renderLogin = Component.renderProp(LoginField);

const VALIDATION_PATH = path.properties.connection.certificateValidationDisabled;

const ValidationField = () => {
  const disabled = Form.useFieldValue(VALIDATION_PATH);
  return (
    <Form.CheckboxField
      path={VALIDATION_PATH}
      label="Validate server certificate"
      inverted
      helpText="Turn off only to troubleshoot. The Driver then accepts any server certificate."
      status={disabled ? "warning" : undefined}
    />
  );
};

const HEADER = ["Device", "Connect", "OPC UA server"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "OPC UA server",
  "aria-label": "Name",
} as const;

const ENDPOINT_INPUT_PROPS = {
  placeholder: "opc.tcp://localhost:4840",
  autoFocus: true,
} as const;

const PATH_INPUT_PROPS = { placeholder: "Path on the Driver computer" } as const;

const PASSWORD_INPUT_PROPS = { type: "password", placeholder: "None" } as const;

const TIMING_BOUNDS = { lower: TimeSpan.MILLISECOND };
