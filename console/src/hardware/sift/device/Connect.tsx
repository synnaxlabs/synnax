// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/sift/device/Connect.css";

import { type device, type rack } from "@synnaxlabs/client";
import {
  Button,
  Device,
  Flex,
  type Flux,
  Form,
  Input,
  Nav,
  Rack,
  Status,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import {
  MAKE,
  type Make,
  MODEL,
  type Properties,
  ZERO_PROPERTIES,
} from "@/hardware/sift/device/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "connectSiftDevice";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Sift.Connect",
  icon: "Logo.Sift",
  location: "modal",
  window: { resizable: false, size: { height: 400, width: 600 }, navTop: true },
};

const useForm = Device.createForm<Properties, Make>();

const INITIAL_VALUES: device.Device<Properties, Make> = {
  key: "",
  name: "",
  make: MAKE,
  model: MODEL,
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateArgs<
  Device.RetrieveQuery,
  typeof Device.formSchema,
  Device.FluxSubStore
>) => {
  let uri = get<string>("properties.uri").value;
  if (uri.startsWith("https://")) uri = uri.slice(8);
  else if (uri.startsWith("http://")) uri = uri.slice(7);
  if (!uri.includes(":")) uri = `${uri}:443`;
  set("location", uri);
  set("properties.uri", uri);
  return true;
};

const beforeSave = async ({
  get,
  set,
}: Flux.FormBeforeSaveParams<
  Device.RetrieveQuery,
  typeof Device.formSchema,
  Device.FluxSubStore
>) => {
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Connected to Sift",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { notifyOnChange: false, markTouched: false });
  return true;
};

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: { key: layoutKey === CONNECT_LAYOUT_TYPE ? "" : layoutKey },
    initialValues: INITIAL_VALUES,
    beforeValidate,
    beforeSave,
    afterSave: useCallback(() => onClose(), [onClose]),
  });
  return (
    <Flex.Box align="start" className={CSS.B("sift-connect")} justify="center" grow>
      <Flex.Box className={CSS.B("content")} grow gap="small">
        <Form.Form<typeof Device.formSchema> {...form}>
          <Form.TextField
            inputProps={{
              autoFocus: true,
              level: "h2",
              placeholder: "Name",
              variant: "text",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect from" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
          <Form.Field<string> path="properties.uri" label="gRPC API URL">
            {(p) => <Input.Text placeholder="grpc-api.siftstack.com:443" {...p} />}
          </Form.Field>
          <Form.Field<string> path="properties.apiKey" label="API key">
            {(p) => <Input.Text placeholder="Your API Key" type="password" {...p} />}
          </Form.Field>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
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
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
