// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/device/Connect.css";

import { type device, type rack, status, TimeSpan } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep, json } from "@synnaxlabs/x";
import { useState } from "react";

import {
  type APIKeyAuthConfigSendAs,
  type AuthType,
  type Device,
  type HealthCheckMethod,
  INTEGRATION,
  type Properties,
  SCHEMAS,
  ZERO_AUTH_CONFIGS,
  ZERO_PROPERTIES,
  ZERO_RESPONSE,
} from "@/feature/http/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/feature/http/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Form as PlatformForm } from "@/platform/form";

const INITIAL_VALUES: Device = {
  key: "",
  name: "HTTP server",
  make: "http",
  model: "HTTP server",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);
// Longer than the usual request timeout to allow for negotiating an SSL handshake.
const TEST_CONNECTION_REQUEST_TIMEOUT_MS = TimeSpan.seconds(5).milliseconds;

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: get<rack.Key>("rack").value,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  const props = get<Properties>("properties").value;
  const host = get<string>("location").value;
  const protocol = props.secure ? "https://" : "http://";
  const connection = {
    base_url: `${protocol}${host}`,
    timeout_ms: TEST_CONNECTION_REQUEST_TIMEOUT_MS,
    verify_ssl: props.verifySsl,
    auth: props.auth,
  };
  const healthCheck = props.healthCheck;
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection, healthCheck },
  });
  if (state.variant === "error") throw status.toError(state);
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    name: get<string>("name").value,
    message: "Device connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { markTouched: false });
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
    saveAsync,
    status: stat,
    variant,
  } = useForm({
    query: deviceKey == null ? null : { key: deviceKey },
    initialValues: { ...INITIAL_VALUES, rack: defaultRack ?? 0 },
    beforeSave,
    afterSave: ({ get }) => {
      if (deviceKey == null) capture("device_connected", { integration: "http" });
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
    name: "HTTP",
    rack: rackKey,
  });

  const { set: setField } = form;

  const authType = Form.useFieldValue<
    AuthType,
    AuthType,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.auth.type", { ctx: form });

  const sendAs = Form.useFieldValue<string, string, PDevice.FormSchema<typeof SCHEMAS>>(
    "properties.auth.sendAs",
    { ctx: form, optional: true },
  );

  const healthCheckMethod = Form.useFieldValue<
    HealthCheckMethod,
    HealthCheckMethod,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.healthCheck.method", { ctx: form });

  const validateResponse = Form.useFieldValue<
    boolean,
    boolean,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.healthCheck.validateResponse", { ctx: form });

  const expectedValueType = Form.useFieldValue<
    json.PrimitiveType,
    json.PrimitiveType,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.healthCheck.response.expectedValueType", {
    ctx: form,
    optional: true,
  });

  const secure = Form.useFieldValue<
    boolean,
    boolean,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.secure", { ctx: form });

  const [tab, setTab] = useState("authentication");

  const handleValidateResponseChange = (value: boolean) => {
    setField("properties.healthCheck.validateResponse", value);
    setField("properties.healthCheck.response", value ? ZERO_RESPONSE : undefined);
  };

  const renderExpectedValueType = ({
    onChange,
    ...rest
  }: SelectExpectedValueTypeProps & {
    onChange: (v: json.PrimitiveType) => void;
  }) => {
    const handleChange = (value: json.PrimitiveType) => {
      setField(
        "properties.healthCheck.response.expectedValue",
        json.ZERO_PRIMITIVES[value],
      );
      onChange(value);
    };
    return <SelectExpectedValueType {...rest} onChange={handleChange} />;
  };

  const renderAuthType = ({
    onChange,
    ...rest
  }: SelectAuthTypeProps & { onChange: (v: AuthType) => void }) => {
    const handleChange = (value: AuthType) => {
      setField("properties.auth", ZERO_AUTH_CONFIGS[value]);
      onChange(value);
    };
    return <SelectAuthType {...rest} onChange={handleChange} />;
  };

  const renderHealthCheckMethod = ({
    onChange,
    ...rest
  }: Select.ButtonsProps<HealthCheckMethod> & {
    onChange: (v: HealthCheckMethod) => void;
  }) => {
    const handleChange = (method: HealthCheckMethod) => {
      onChange(method);
      setField("properties.healthCheck.body", method === "POST" ? "" : undefined);
    };
    return <SelectHealthCheckMethod {...rest} onChange={handleChange} />;
  };

  const renderSendAs = ({
    onChange,
    ...rest
  }: SelectSendAsProps & { onChange: (v: APIKeyAuthConfigSendAs) => void }) => {
    const handleChange = (value: APIKeyAuthConfigSendAs) => {
      if (value === "header") setField("properties.auth.header", "");
      else setField("properties.auth.parameter", "");
      onChange(value);
    };
    return <SelectSendAs {...rest} onChange={handleChange} />;
  };

  const handleConnect = async () => {
    if (await saveAsync()) return;
    const errorPaths = form
      .getStatuses()
      .flatMap(({ key, variant }) => (variant === "error" && key != null ? [key] : []));
    const holdsError = (tabKey: string) =>
      TAB_PATHS[tabKey].some((tabPath) =>
        errorPaths.some((path) => deep.pathsMatch(path, tabPath)),
      );
    if (holdsError(tab)) return;
    const next = Object.keys(TAB_PATHS).find(holdsError);
    if (next != null) setTab(next);
  };
  return (
    <Modal.Frame className={CSS.B("http-connect")}>
      <Modal.Header icon={<Icon.Logo.HTTP />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large" justify="start">
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
                path="location"
                label="Host"
                padHelpText={false}
                inputProps={HOST_INPUT_PROPS}
              />
              <Form.CheckboxField
                path="properties.secure"
                label="HTTPS"
                padHelpText={false}
              />
              {secure && (
                <Form.CheckboxField
                  path="properties.verifySsl"
                  label="Verify certificate"
                  padHelpText={false}
                />
              )}
              <PlatformDevice.RackField integration={INTEGRATION} />
            </Form.Section>
          </Form.Sections>
          <Tabs.Frame
            value={tab}
            onChange={setTab}
            className={CSS.BE("http-connect", "tabs")}
            empty={false}
            gap="large"
          >
            <Tabs.Selector variant="chip">
              <Tabs.Tab itemKey="authentication">Authentication</Tabs.Tab>
              <Tabs.Tab itemKey="health_check">Health check</Tabs.Tab>
              <Tabs.Tab itemKey="advanced">Advanced</Tabs.Tab>
            </Tabs.Selector>
            <Tabs.Content itemKey="authentication">
              <Form.Sections>
                <Form.Section>
                  <Form.Field<AuthType>
                    path="properties.auth.type"
                    label="Type"
                    padHelpText={false}
                  >
                    {renderAuthType}
                  </Form.Field>
                  {authType === "bearer" && (
                    <Form.TextField
                      path="properties.auth.token"
                      label="Token"
                      padHelpText={false}
                      inputProps={AUTH_TOKEN_INPUT_PROPS}
                    />
                  )}
                  {authType === "api_key" && (
                    <>
                      <Form.Field<APIKeyAuthConfigSendAs>
                        path="properties.auth.sendAs"
                        label="Send as"
                        padHelpText={false}
                      >
                        {renderSendAs}
                      </Form.Field>
                      {sendAs === "query_param" ? (
                        <Form.TextField
                          path="properties.auth.parameter"
                          label="Parameter"
                          padHelpText={false}
                          inputProps={AUTH_PARAM_INPUT_PROPS}
                        />
                      ) : (
                        <Form.TextField
                          path="properties.auth.header"
                          label="Header"
                          padHelpText={false}
                          inputProps={AUTH_HEADER_INPUT_PROPS}
                        />
                      )}
                      <Form.TextField
                        path="properties.auth.key"
                        label="Key"
                        padHelpText={false}
                        inputProps={AUTH_KEY_INPUT_PROPS}
                      />
                    </>
                  )}
                  {authType === "basic" && (
                    <>
                      <Form.TextField
                        path="properties.auth.username"
                        label="Username"
                        padHelpText={false}
                        inputProps={AUTH_USERNAME_INPUT_PROPS}
                      />
                      <Form.TextField
                        path="properties.auth.password"
                        label="Password"
                        padHelpText={false}
                        inputProps={AUTH_PASSWORD_INPUT_PROPS}
                      />
                    </>
                  )}
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
            <Tabs.Content itemKey="health_check">
              <Form.Sections>
                <Form.Section title="Request">
                  <Form.Field<HealthCheckMethod>
                    path="properties.healthCheck.method"
                    label="Method"
                    padHelpText={false}
                  >
                    {renderHealthCheckMethod}
                  </Form.Field>
                  <Form.TextField
                    path="properties.healthCheck.path"
                    label="Path"
                    padHelpText={false}
                    inputProps={HEALTH_PATH_INPUT_PROPS}
                  />
                  {healthCheckMethod === "POST" && (
                    <Form.TextField
                      path="properties.healthCheck.body"
                      label="Body"
                      padHelpText={false}
                      inputProps={HEALTH_BODY_INPUT_PROPS}
                    />
                  )}
                  <PlatformForm.KeyValueEditor
                    path="properties.healthCheck.headers"
                    resourceName="header"
                    label="Headers"
                    keyField="name"
                    keyPlaceholder="Content-Type"
                    valuePlaceholder="application/json"
                  />
                  <PlatformForm.KeyValueEditor
                    path="properties.healthCheck.queryParams"
                    resourceName="parameter"
                    label="Query parameters"
                    keyField="parameter"
                    keyPlaceholder="limit"
                    valuePlaceholder="100"
                  />
                </Form.Section>
                <Form.Section title="Response">
                  <Form.CheckboxField
                    path="properties.healthCheck.validateResponse"
                    label="Validate body"
                    padHelpText={false}
                    onChange={handleValidateResponseChange}
                  />
                  {validateResponse && (
                    <>
                      <Form.TextField
                        path="properties.healthCheck.response.pointer"
                        label="JSON pointer"
                        padHelpText={false}
                        inputProps={HEALTH_POINTER_INPUT_PROPS}
                      />
                      <Form.Field<json.PrimitiveType>
                        path="properties.healthCheck.response.expectedValueType"
                        label="Value type"
                        padHelpText={false}
                      >
                        {renderExpectedValueType}
                      </Form.Field>
                      {expectedValueType === "string" && (
                        <Form.TextField
                          path="properties.healthCheck.response.expectedValue"
                          label="Expected value"
                          padHelpText={false}
                          inputProps={HEALTH_EXPECTED_STRING_INPUT_PROPS}
                        />
                      )}
                      {expectedValueType === "number" && (
                        <Form.NumericField
                          path="properties.healthCheck.response.expectedValue"
                          label="Expected value"
                          padHelpText={false}
                        />
                      )}
                      {expectedValueType === "boolean" && (
                        <Form.CheckboxField
                          path="properties.healthCheck.response.expectedValue"
                          label="Expected value"
                          padHelpText={false}
                        />
                      )}
                    </>
                  )}
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
            <Tabs.Content itemKey="advanced">
              <Form.Sections>
                <Form.Section>
                  <Form.NumericField
                    path="properties.timeoutMs"
                    label="Timeout"
                    padHelpText={false}
                    inputProps={TIMEOUT_INPUT_PROPS}
                  />
                  <Form.NumericField
                    path="properties.maxConcurrentRequests"
                    label="Concurrent requests"
                    padHelpText={false}
                    inputProps={MAX_CONCURRENT_REQUESTS_INPUT_PROPS}
                  />
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
          </Tabs.Frame>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => void handleConnect()}
        onBack={onBack}
      />
    </Modal.Frame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(Connect, INTEGRATION);

const HEADER = ["Device", "Connect", "HTTP server"];

// The paths each tab edits, so a failed Connect can open the tab with the error.
const TAB_PATHS: Record<string, string[]> = {
  authentication: ["properties.auth"],
  health_check: ["properties.healthCheck"],
  advanced: ["properties.timeoutMs", "properties.maxConcurrentRequests"],
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "HTTP server",
  "aria-label": "Name",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "www.example.com" } as const;

const TIMEOUT_INPUT_PROPS = { endContent: "ms", style: { width: "23rem" } } as const;

const MAX_CONCURRENT_REQUESTS_INPUT_PROPS = {
  bounds: { lower: 1, upper: Infinity },
  style: { width: "23rem" },
} as const;

const AUTH_TOKEN_INPUT_PROPS = {
  placeholder: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  type: "password",
} as const;

const AUTH_HEADER_INPUT_PROPS = { placeholder: "X-API-Key" } as const;

const AUTH_PARAM_INPUT_PROPS = { placeholder: "key" } as const;

const AUTH_KEY_INPUT_PROPS = {
  placeholder: "sk_live_51N8...",
  type: "password",
} as const;

const AUTH_USERNAME_INPUT_PROPS = { placeholder: "user@example.com" } as const;

const AUTH_PASSWORD_INPUT_PROPS = { type: "password" } as const;

interface SelectAuthTypeProps extends Select.ButtonsProps<AuthType> {}

const SelectAuthType = (props: SelectAuthTypeProps) => (
  <Select.Buttons<AuthType> {...props}>
    <Select.Item<AuthType> itemKey="none">None</Select.Item>
    <Select.Item<AuthType>
      itemKey="bearer"
      tooltip={authBearerTooltip}
      tooltipLocation="top"
    >
      Bearer token
    </Select.Item>
    <Select.Item<AuthType>
      itemKey="api_key"
      tooltip="Sends your API key as a header or query parameter"
      tooltipLocation="top"
    >
      API key
    </Select.Item>
    <Select.Item<AuthType>
      itemKey="basic"
      tooltip={authBasicTooltip}
      tooltipLocation="top"
    >
      Basic
    </Select.Item>
  </Select.Buttons>
);

interface SelectSendAsProps extends Select.ButtonsProps<APIKeyAuthConfigSendAs> {}

const SelectSendAs = (props: SelectSendAsProps) => (
  <Select.Buttons<APIKeyAuthConfigSendAs> {...props}>
    <Select.Item<APIKeyAuthConfigSendAs> itemKey="header">Header</Select.Item>
    <Select.Item<APIKeyAuthConfigSendAs> itemKey="query_param">
      Query parameter
    </Select.Item>
  </Select.Buttons>
);

const HEALTH_PATH_INPUT_PROPS = { placeholder: "/health" } as const;

const HEALTH_BODY_INPUT_PROPS = { placeholder: '{"check": "ping"}' } as const;

const HEALTH_POINTER_INPUT_PROPS = { placeholder: "/status" } as const;

const HEALTH_EXPECTED_STRING_INPUT_PROPS = { placeholder: "ok" } as const;

const SelectHealthCheckMethod = (props: Select.ButtonsProps<HealthCheckMethod>) => (
  <Select.Buttons<HealthCheckMethod> {...props}>
    <Select.Item<HealthCheckMethod> itemKey="GET">GET</Select.Item>
    <Select.Item<HealthCheckMethod> itemKey="POST">POST</Select.Item>
  </Select.Buttons>
);

interface SelectExpectedValueTypeProps extends Select.ButtonsProps<json.PrimitiveType> {}

const SelectExpectedValueType = (props: SelectExpectedValueTypeProps) => (
  <Select.Buttons<json.PrimitiveType> {...props}>
    <Select.Item<json.PrimitiveType> itemKey="string">String</Select.Item>
    <Select.Item<json.PrimitiveType> itemKey="number">Number</Select.Item>
    <Select.Item<json.PrimitiveType> itemKey="boolean">Boolean</Select.Item>
    <Select.Item<json.PrimitiveType> itemKey="null">Null</Select.Item>
  </Select.Buttons>
);

const authBearerTooltip = (
  <Text.Text level="small" color={11}>
    Sends{" "}
    <Text.Text level="small" variant="code" color={11}>
      Authorization: Bearer {"<token>"}
    </Text.Text>{" "}
    header
  </Text.Text>
);

const authBasicTooltip = (
  <Text.Text level="small" color={11}>
    Sends base64-encoded credentials via the{" "}
    <Text.Text level="small" variant="code" color={11}>
      Authorization
    </Text.Text>{" "}
    header
  </Text.Text>
);
