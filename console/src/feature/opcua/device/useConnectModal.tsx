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
import { Modal } from "@synnaxlabs/lyra/modal";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";

import { retrieveScanTask } from "@/feature/opcua/device/retrieveScanTask";
import {
  SelectSecurityMode,
  type SelectSecurityModeProps,
} from "@/feature/opcua/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/feature/opcua/device/SelectSecurityPolicy";
import {
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
import { TEST_CONNECTION_COMMAND_TYPE } from "@/feature/opcua/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { FS } from "@/platform/fs";

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

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const scanTask = await retrieveScanTask(client, get<rack.Key>("rack").value);
  const scanStatus = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection: get("properties.connection").value },
  });
  if (scanStatus.variant === "error") throw status.toError(scanStatus);
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

export const Connect = ({
  deviceKey,
  rack: defaultRack,
  close,
  onBack,
}: PlatformDevice.ConnectProps) => {
  const { capture } = Analytics.use();
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
          <Form.TextField path="name" showLabel={false} inputProps={NAME_INPUT_PROPS} />
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
                    {({ value, onChange }) => (
                      <SelectSecurityPolicy value={value} onChange={onChange} />
                    )}
                  </Form.Field>
                  <Form.Field<string>
                    path="properties.connection.clientCertificate"
                    label="Client certificate"
                    padHelpText={false}
                  >
                    {clientCertField}
                  </Form.Field>
                  <Form.Field<string>
                    path="properties.connection.clientPrivateKey"
                    label="Client private key"
                    padHelpText={false}
                  >
                    {clientPrivateKeyField}
                  </Form.Field>
                  <Form.Field<string>
                    path="properties.connection.serverCertificate"
                    label="Server certificate"
                    padHelpText={false}
                  >
                    {serverCertField}
                  </Form.Field>
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

const clientCertField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a client certificate" />
));

const clientPrivateKeyField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a client private key" />
));

const serverCertField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a server certificate" />
));

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

const PASSWORD_INPUT_PROPS = { type: "password" } as const;
