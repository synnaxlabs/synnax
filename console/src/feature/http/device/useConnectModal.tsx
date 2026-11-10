// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/device/Connect.css";

import { type device, status, TimeSpan } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
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
  INTEGRATION_KEY,
  type Response,
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

const path = deep.path<Device>();
const { auth, healthCheck } = path.properties;
const apiKey = deep.variant(auth, "type", "api_key");
const bearer = deep.variant(auth, "type", "bearer");
const basic = deep.variant(auth, "type", "basic");
const healthBody = deep.variant(healthCheck, "method", "POST").body;
const response = deep.variant(healthCheck, "validateResponse", true).response;
const apiKeyHeader = deep.variant(apiKey, "sendAs", "header").header;
const apiKeyParameter = deep.variant(apiKey, "sendAs", "query_param").parameter;

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
    rack: get(path.rack).value,
    one: true,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  const props = get(path.properties).value;
  const host = get(path.location).value;
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
    name: get(path.name).value,
    message: "Device connected",
    variant: "success",
    details: {
      rack: get(path.rack).value,
      connected: true,
    },
  });
  set(path.status, devStatus, { markTouched: false });
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
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "HTTP",
    rack: rackKey,
  });

  const { set: setField } = form;

  const authType = Form.useFieldValue(auth.type, { ctx: form });

  const sendAs = Form.useFieldValue(apiKey.sendAs, { ctx: form, optional: true });

  const healthCheckMethod = Form.useFieldValue(healthCheck.method, { ctx: form });

  const validateResponse = Form.useFieldValue(healthCheck.validateResponse, {
    ctx: form,
  });

  const expectedValueType = Form.useFieldValue(response.expectedValueType, {
    ctx: form,
    optional: true,
  });

  const secure = Form.useFieldValue(path.properties.secure, { ctx: form });

  const [tab, setTab] = useState("authentication");

  const handleValidateResponseChange = (value: boolean) => {
    setField(healthCheck.validateResponse, value);
    setField(
      deep.widen<Response | undefined>(response),
      value ? ZERO_RESPONSE : undefined,
    );
  };

  const renderExpectedValueType = ({
    onChange,
    ...rest
  }: SelectExpectedValueTypeProps & {
    onChange: (v: json.PrimitiveType) => void;
  }) => {
    const handleChange = (value: json.PrimitiveType) => {
      setField(response.expectedValue, json.ZERO_PRIMITIVES[value]);
      onChange(value);
    };
    return <SelectExpectedValueType {...rest} onChange={handleChange} />;
  };

  const renderAuthType = ({
    onChange,
    ...rest
  }: SelectAuthTypeProps & { onChange: (v: AuthType) => void }) => {
    const handleChange = (value: AuthType) => {
      setField(auth, ZERO_AUTH_CONFIGS[value]);
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
      setField(healthBody, method === "POST" ? "" : undefined);
    };
    return <SelectHealthCheckMethod {...rest} onChange={handleChange} />;
  };

  const renderSendAs = ({
    onChange,
    ...rest
  }: SelectSendAsProps & { onChange: (v: APIKeyAuthConfigSendAs) => void }) => {
    const handleChange = (value: APIKeyAuthConfigSendAs) => {
      if (value === "header") setField(apiKeyHeader, "");
      else setField(apiKeyParameter, "");
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
    <PlatformDevice.ConnectFrame className={CSS.B("http-connect")}>
      <Modal.Header icon={<Icon.Logo.HTTP />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large" justify="start">
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
                path={path.location}
                label="Host"
                padHelpText={false}
                inputProps={HOST_INPUT_PROPS}
              />
              <Form.CheckboxField
                path={path.properties.secure}
                label="HTTPS"
                padHelpText={false}
              />
              {secure && (
                <Form.CheckboxField
                  path={path.properties.verifySsl}
                  label="Verify certificate"
                  padHelpText={false}
                />
              )}
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
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
                  <Form.Field path={auth.type} label="Type" padHelpText={false}>
                    {renderAuthType}
                  </Form.Field>
                  {authType === "bearer" && (
                    <Form.TextField
                      path={bearer.token}
                      label="Token"
                      padHelpText={false}
                      inputProps={AUTH_TOKEN_INPUT_PROPS}
                    />
                  )}
                  {authType === "api_key" && (
                    <>
                      <Form.Field
                        path={apiKey.sendAs}
                        label="Send as"
                        padHelpText={false}
                      >
                        {renderSendAs}
                      </Form.Field>
                      {sendAs === "query_param" ? (
                        <Form.TextField
                          path={apiKeyParameter}
                          label="Parameter"
                          padHelpText={false}
                          inputProps={AUTH_PARAM_INPUT_PROPS}
                        />
                      ) : (
                        <Form.TextField
                          path={apiKeyHeader}
                          label="Header"
                          padHelpText={false}
                          inputProps={AUTH_HEADER_INPUT_PROPS}
                        />
                      )}
                      <Form.TextField
                        path={apiKey.key}
                        label="Key"
                        padHelpText={false}
                        inputProps={AUTH_KEY_INPUT_PROPS}
                      />
                    </>
                  )}
                  {authType === "basic" && (
                    <Flex.Box x>
                      <Form.TextField
                        path={basic.username}
                        label="Username"
                        padHelpText={false}
                        inputProps={AUTH_USERNAME_INPUT_PROPS}
                      />
                      <Form.TextField
                        path={basic.password}
                        label="Password"
                        padHelpText={false}
                        inputProps={AUTH_PASSWORD_INPUT_PROPS}
                      />
                    </Flex.Box>
                  )}
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
            <Tabs.Content itemKey="health_check">
              <Form.Sections>
                <Form.Section title="Request">
                  <Form.Field
                    path={healthCheck.method}
                    label="Method"
                    padHelpText={false}
                  >
                    {renderHealthCheckMethod}
                  </Form.Field>
                  <Form.TextField
                    path={healthCheck.path}
                    label="Path"
                    padHelpText={false}
                    inputProps={HEALTH_PATH_INPUT_PROPS}
                  />
                  {healthCheckMethod === "POST" && (
                    <Form.TextField
                      path={healthBody}
                      label="Body"
                      padHelpText={false}
                      inputProps={HEALTH_BODY_INPUT_PROPS}
                    />
                  )}
                  <PlatformForm.KeyValueEditor
                    path={deep.present(healthCheck.headers)}
                    resourceName="header"
                    label="Headers"
                    keyField="name"
                    keyPlaceholder="Content-Type"
                    valuePlaceholder="application/json"
                  />
                  <PlatformForm.KeyValueEditor
                    path={deep.present(healthCheck.queryParams)}
                    resourceName="parameter"
                    label="Query parameters"
                    keyField="parameter"
                    keyPlaceholder="limit"
                    valuePlaceholder="100"
                  />
                </Form.Section>
                <Form.Section title="Response">
                  <Form.CheckboxField
                    path={healthCheck.validateResponse}
                    label="Validate body"
                    padHelpText={false}
                    onChange={handleValidateResponseChange}
                  />
                  {validateResponse && (
                    <>
                      <Form.TextField
                        path={response.pointer}
                        label="JSON pointer"
                        padHelpText={false}
                        inputProps={HEALTH_POINTER_INPUT_PROPS}
                      />
                      <Form.Field
                        path={response.expectedValueType}
                        label="Value type"
                        padHelpText={false}
                      >
                        {renderExpectedValueType}
                      </Form.Field>
                      {expectedValueType === "string" && (
                        <Form.TextField
                          path={
                            deep.variant(response, "expectedValueType", "string")
                              .expectedValue
                          }
                          label="Expected value"
                          padHelpText={false}
                          inputProps={HEALTH_EXPECTED_STRING_INPUT_PROPS}
                        />
                      )}
                      {expectedValueType === "number" && (
                        <Form.NumericField
                          path={
                            deep.variant(response, "expectedValueType", "number")
                              .expectedValue
                          }
                          label="Expected value"
                          padHelpText={false}
                        />
                      )}
                      {expectedValueType === "boolean" && (
                        <Form.CheckboxField
                          path={
                            deep.variant(response, "expectedValueType", "boolean")
                              .expectedValue
                          }
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
                  <Form.TimeSpanField
                    path={path.properties.timeoutMs}
                    label="Timeout"
                    unit={TimeSpan.MILLISECOND}
                    padHelpText={false}
                  />
                  <Form.NumericField
                    path={path.properties.maxConcurrentRequests}
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
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const HEADER = ["Device", "Connect", "HTTP server"];

// The paths each tab edits, so a failed Connect can open the tab with the error.
const TAB_PATHS: Record<string, string[]> = {
  authentication: [String(auth)],
  health_check: [String(healthCheck)],
  advanced: [
    String(path.properties.timeoutMs),
    String(path.properties.maxConcurrentRequests),
  ],
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "HTTP server",
  "aria-label": "Name",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "www.example.com" } as const;

const MAX_CONCURRENT_REQUESTS_INPUT_PROPS = {
  bounds: { lower: 1, upper: Infinity },
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
