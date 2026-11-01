// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Connect.css";

import { type device, type rack, status, TimeSpan } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { useState } from "react";

import {
  DriverCertificate,
  ServerCertificate,
} from "@/feature/opcua/device/Certificate";
import { retrieveScanTask } from "@/feature/opcua/device/retrieveScanTask";
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
  INTEGRATION,
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
} from "@/feature/opcua/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

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
>) => set("location", get("properties.connection.endpoint").value);

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
    const scanTask = await retrieveScanTask(client, get<rack.Key>("rack").value);
    const scanStatus = await scanTask.executeCommandSync({
      type: TEST_CONNECTION_COMMAND_TYPE,
      timeout: TEST_CONNECTION_TIMEOUT,
      args: { connection: get("properties.connection").value },
    });
    if (scanStatus.variant === "error") {
      const data = scanStatus.details?.data;
      if (data != null && "serverCertificate" in data && data.serverCertificate != null)
        setServerCertificateOffer(data.serverCertificate);
      throw status.toError(scanStatus);
    }
    // Since we just scanned successfully, we create a default healthy status for the
    // device that can then be overwritten by the scanner if we lose connection.
    const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
      name: get<string>("name").value,
      message: "Server connected",
      variant: "success",
      details: {
        rack: get<rack.Key>("rack").value,
        device: get<device.Key>("key").value,
      },
    });
    set("status", devStatus, { notifyOnChange: false, markTouched: false });
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
      close(get<device.Key>("key").value);
    },
  });
  const rackKey = Form.useFieldValue<
    rack.Key,
    rack.Key,
    PDevice.FormSchema<typeof SCHEMAS>
  >("rack", { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION,
    name: "OPC UA",
    rack: rackKey,
  });

  const { get: getField, set: setField } = form;
  const hasSecurity =
    Form.useFieldValue<SecurityMode, SecurityMode, PDevice.FormSchema<typeof SCHEMAS>>(
      "properties.connection.securityMode",
      { ctx: form },
    ) != NO_SECURITY_MODE;
  const clientCertificate = Form.useFieldValue<
    string | undefined,
    string | undefined,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.connection.clientCertificate", { ctx: form });
  const clientPrivateKey = Form.useFieldValue<
    string | undefined,
    string | undefined,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.connection.clientPrivateKey", { ctx: form });
  const serverCertificate =
    Form.useFieldValue<
      string | undefined,
      string | undefined,
      PDevice.FormSchema<typeof SCHEMAS>
    >("properties.connection.serverCertificate", { ctx: form }) ?? "";
  // Empty paths select the Driver certificate, so a device that names files has chosen
  // them until the user picks again.
  const hasFiles = (clientCertificate ?? "") !== "" || (clientPrivateKey ?? "") !== "";
  const source = clientCertificateSource ?? (hasFiles ? "files" : "driver");

  const handleSourceChange = (next: ClientCertificateSource) => {
    setClientCertificateSource(next);
    if (next !== "driver") return;
    setField("properties.connection.clientCertificate", "");
    setField("properties.connection.clientPrivateKey", "");
  };

  const handleTrust = (offer: CertificateInfo) => {
    setField("properties.connection.serverCertificate", offer.pem);
    save();
  };

  const renderSecurityMode = ({
    onChange,
    ...rest
  }: SelectSecurityModeProps & { onChange: (v: SecurityMode) => void }) => {
    const handleChange = (mode: SecurityMode) => {
      onChange(mode);
      const policyPath = "properties.connection.securityPolicy";
      // Security mode None pairs only with policy None, and every other mode with a
      // policy other than None.
      if (mode === NO_SECURITY_MODE) setField(policyPath, NO_SECURITY_POLICY);
      else if (getField<SecurityPolicy>(policyPath).value === NO_SECURITY_POLICY)
        setField(policyPath, BASIC256_SHA256_SECURITY_POLICY);
    };
    return <SelectSecurityMode {...rest} onChange={handleChange} />;
  };

  return (
    <Modal.Frame className={CSS.B("opc-connect")}>
      <Modal.Header icon={<Icon.Logo.OPCUA />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path="name"
            showLabel={false}
            padHelpText={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections>
            <Form.Section title="Connection">
              <Form.TextField
                path="properties.connection.endpoint"
                label="Endpoint"
                padHelpText={false}
                inputProps={ENDPOINT_INPUT_PROPS}
              />
              <PlatformDevice.RackField integration={INTEGRATION} />
            </Form.Section>
            <Form.Section title="Authentication">
              <Form.TextField
                path="properties.connection.username"
                label="Username"
                padHelpText={false}
                inputProps={USERNAME_INPUT_PROPS}
              />
              <Form.TextField
                path="properties.connection.password"
                label="Password"
                padHelpText={false}
                inputProps={PASSWORD_INPUT_PROPS}
              />
            </Form.Section>
            <Form.Section title="Security">
              <Form.Field<SecurityMode>
                path="properties.connection.securityMode"
                label="Mode"
                padHelpText={false}
              >
                {renderSecurityMode}
              </Form.Field>
              {hasSecurity && (
                <>
                  <Form.Field<SecurityPolicy>
                    path="properties.connection.securityPolicy"
                    label="Policy"
                    padHelpText={false}
                  >
                    {renderSecurityPolicy}
                  </Form.Field>
                  <Input.Item label="Client certificate">
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
                  </Input.Item>
                  {source === "files" && (
                    <>
                      <Form.TextField
                        path="properties.connection.clientCertificate"
                        label="Certificate file"
                        padHelpText={false}
                        inputProps={PATH_INPUT_PROPS}
                      />
                      <Form.TextField
                        path="properties.connection.clientPrivateKey"
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
    </Modal.Frame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(Connect, INTEGRATION);

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

const VALIDATION_PATH = "properties.connection.certificateValidationDisabled";

const ValidationField = () => {
  const disabled = Form.useFieldValue<boolean>(VALIDATION_PATH);
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

const USERNAME_INPUT_PROPS = { placeholder: "Anonymous" } as const;

const PATH_INPUT_PROPS = { placeholder: "Path on the Driver computer" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;
