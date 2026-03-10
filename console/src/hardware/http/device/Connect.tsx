// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/device/Connect.css";

import { type rack } from "@synnaxlabs/client";
import {
  Button,
  Device as PDevice,
  Divider,
  Flex,
  Form,
  Nav,
  Rack,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import {
  type APIKeyAuthConfigSendAs,
  type AuthType,
  type Device,
  SCHEMAS,
  ZERO_AUTH_CONFIGS,
  ZERO_PROPERTIES,
} from "@/hardware/http/device/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "configureHTTPServer";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Server.Connect",
  icon: "Logo.HTTP",
  location: "modal",
  window: { resizable: false, size: { height: 530, width: 700 }, navTop: true },
};

const INITIAL_VALUES: Device = {
  key: "",
  name: "HTTP Server",
  make: "http",
  model: "HTTP server",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS);

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: { key: layoutKey === CONNECT_LAYOUT_TYPE ? "" : layoutKey },
    initialValues: INITIAL_VALUES,
    afterSave: useCallback(() => onClose(), [onClose]),
  });

  const authType = Form.useFieldValue<AuthType, AuthType, typeof PDevice.formSchema>(
    "properties.auth.type",
    { ctx: form },
  );

  const sendAs = Form.useFieldValue<string, string, typeof PDevice.formSchema>(
    "properties.auth.sendAs",
    { ctx: form, optional: true },
  );

  const renderAuthType = useCallback(
    ({
      onChange,
      ...rest
    }: SelectAuthTypeProps & { onChange: (v: AuthType) => void }) => {
      const handleChange = (value: AuthType) => {
        form.set("properties.auth", ZERO_AUTH_CONFIGS[value]);
        onChange(value);
      };
      return <SelectAuthType {...rest} onChange={handleChange} />;
    },
    [form.set],
  );

  const renderSendAs = useCallback(
    ({
      onChange,
      ...rest
    }: SelectSendAsProps & { onChange: (v: APIKeyAuthConfigSendAs) => void }) => {
      const handleChange = (value: APIKeyAuthConfigSendAs) => {
        if (value === "header") form.set("properties.auth.header", "");
        else form.set("properties.auth.parameter", "");
        onChange(value);
      };
      return <SelectSendAs {...rest} onChange={handleChange} />;
    },
    [form.set],
  );

  return (
    <Flex.Box grow className={CSS.B("http-connect")}>
      <Flex.Box className={CSS.B("content")} grow gap="small">
        <Form.Form<typeof PDevice.formSchema> {...form}>
          <Form.TextField path="name" inputProps={NAME_INPUT_PROPS} />
          <Form.Field<rack.Key> path="rack" label="Connect from" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
          <Flex.Box x align="end">
            <Form.TextField
              grow
              path="location"
              label="Host"
              inputProps={HOST_INPUT_PROPS}
            />
            <Form.SwitchField path="properties.secure" label="HTTPS" />
            <Form.SwitchField path="properties.verifySsl" label="Verify SSL" />
          </Flex.Box>
          <Form.NumericField
            path="properties.timeoutMs"
            label="Expected response time"
            inputProps={TIMEOUT_INPUT_PROPS}
          />
          <Divider.Divider x padded="bottom" />
          <Form.Field<AuthType> path="properties.auth.type" label="Authentication">
            {renderAuthType}
          </Form.Field>
          {authType === "bearer" && (
            <Form.TextField
              path="properties.auth.token"
              label="Token"
              inputProps={AUTH_TOKEN_INPUT_PROPS}
            />
          )}
          {authType === "api_key" && (
            <>
              <Form.Field<APIKeyAuthConfigSendAs>
                path="properties.auth.sendAs"
                label="Send as"
              >
                {renderSendAs}
              </Form.Field>
              <Flex.Box x justify="between">
                {sendAs === "query_param" ? (
                  <Form.TextField
                    grow
                    path="properties.auth.parameter"
                    label="Parameter Name"
                    inputProps={AUTH_PARAM_INPUT_PROPS}
                  />
                ) : (
                  <Form.TextField
                    grow
                    path="properties.auth.header"
                    label="Header Name"
                    inputProps={AUTH_HEADER_INPUT_PROPS}
                  />
                )}
                <Form.TextField
                  grow
                  path="properties.auth.key"
                  label="API Key"
                  inputProps={AUTH_KEY_INPUT_PROPS}
                />
              </Flex.Box>
            </>
          )}
          {authType === "basic" && (
            <Flex.Box x justify="between">
              <Form.TextField
                grow
                path="properties.auth.username"
                label="Username"
                inputProps={AUTH_USERNAME_INPUT_PROPS}
              />
              <Form.TextField
                grow
                path="properties.auth.password"
                label="Password"
                inputProps={AUTH_PASSWORD_INPUT_PROPS}
              />
            </Flex.Box>
          )}
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {variant == "success" ? (
            <Triggers.SaveHelpText action="Save" noBar />
          ) : (
            <Status.Summary variant={variant} message={stat.description} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={() => save()}
            variant="filled"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "HTTP Server",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "www.example.com" } as const;

const TIMEOUT_INPUT_PROPS = { endContent: "ms", style: { width: "23rem" } } as const;

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

const SELECT_AUTH_TYPE_DATA: AuthType[] = ["none", "bearer", "api_key", "basic"];

interface SelectAuthTypeProps extends Omit<Select.ButtonsProps<AuthType>, "keys"> {}

const SelectAuthType = (props: SelectAuthTypeProps) => (
  <Select.Buttons<AuthType> {...props} keys={SELECT_AUTH_TYPE_DATA}>
    <Select.Button<AuthType> itemKey="none">None</Select.Button>
    <Select.Button<AuthType>
      itemKey="bearer"
      tooltip={authBearerTooltip}
      tooltipLocation="top"
    >
      Bearer Token
    </Select.Button>
    <Select.Button<AuthType>
      itemKey="api_key"
      tooltip="Sends your API key as a header or query parameter"
      tooltipLocation="top"
    >
      API Key
    </Select.Button>
    <Select.Button<AuthType>
      itemKey="basic"
      tooltip={authBasicTooltip}
      tooltipLocation="top"
    >
      Basic
    </Select.Button>
  </Select.Buttons>
);

const SEND_AS_DATA: APIKeyAuthConfigSendAs[] = ["header", "query_param"];

interface SelectSendAsProps extends Omit<
  Select.ButtonsProps<APIKeyAuthConfigSendAs>,
  "keys"
> {}

const SelectSendAs = (props: SelectSendAsProps) => (
  <Select.Buttons<APIKeyAuthConfigSendAs> {...props} keys={SEND_AS_DATA}>
    <Select.Button<APIKeyAuthConfigSendAs> itemKey="header">Header</Select.Button>
    <Select.Button<APIKeyAuthConfigSendAs> itemKey="query_param">
      Query Parameter
    </Select.Button>
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
