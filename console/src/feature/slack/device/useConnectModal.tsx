// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, status } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Device as PDevice,
  type Flux,
  Form,
  Icon,
  Nav,
  Rack,
  Status,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Device, SCHEMAS, ZERO_PROPERTIES } from "@/feature/slack/device/types";
import { type Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const INITIAL_VALUES: Device = {
  key: "",
  name: "Slack Workspace",
  make: "slack",
  model: "Slack workspace",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS);

// beforeSave marks the device connected. Live token validation via Slack auth.test is
// deferred; a bad token surfaces when the first alert task posts.
const beforeSave = async ({
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema,
  PDevice.FluxSubStore
>) => {
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Workspace connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { markTouched: false });
  return true;
};

export const useConnectModal = Modals.create<PlatformDevice.ConnectParams>(
  ({ deviceKey, close }) => {
    const {
      form,
      save,
      status: stat,
      variant,
    } = useForm({
      query: { key: deviceKey ?? "" },
      initialValues: INITIAL_VALUES,
      beforeSave,
      afterSave: useCallback(() => close(), [close]),
    });

    return (
      <Modals.Frame>
        <Modals.Header icon={<Icon.Logo.Slack />}>Workspace.Connect</Modals.Header>
        <Form.Form<typeof PDevice.formSchema> {...form}>
          <Modals.Body>
            <Form.TextField path="name" inputProps={NAME_INPUT_PROPS} />
            <Form.Field<rack.Key> path="rack" label="Connect from" required>
              {selectRackRenderProp}
            </Form.Field>
            <Form.TextField
              path="properties.botToken"
              label="Bot token"
              inputProps={TOKEN_INPUT_PROPS}
            />
          </Modals.Body>
        </Form.Form>
        <Modals.Footer>
          <Nav.Bar.Start gap="small">
            {variant == "success" ? (
              <Triggers.SaveHelpText action="Connect" noBar />
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
              Connect
            </Button.Button>
          </Nav.Bar.End>
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);

const INITIAL_RACK_QUERY: rack.RetrieveArgs = { integration: "slack" };

const selectRackRenderProp = Component.renderProp(
  (props: Pick<Rack.SelectSingleProps, "value" | "onChange">) => (
    <Rack.SelectSingle {...props} initialQuery={INITIAL_RACK_QUERY} />
  ),
);

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Slack Workspace",
} as const;

const TOKEN_INPUT_PROPS = {
  autoFocus: true,
  type: "password",
  placeholder: "xoxb-your-bot-token",
} as const;
