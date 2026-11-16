// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/sift/device/Connect.css";

import { type device, rack, status, TimeSpan } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";

import {
  type Device,
  INTEGRATION,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/sift/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/feature/sift/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Secret } from "@/platform/secret";

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(15);

const path = deep.path<Device>();

const INITIAL_VALUES: Device = {
  key: "",
  name: "",
  make: "sift",
  model: "sift",
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
>) => set(path.location, get(path.properties.uri).value);

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const rackKey = get(path.rack).value;
  const connection = get(path.properties).value;
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: rackKey,
    one: true,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  // The rack reads the API key once, for this test only.
  const grant = await client.secrets.grant({
    subject: rack.resourceID(rackKey),
    secrets: [connection.apiKey],
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection, grant },
  });
  if (state.variant === "error")
    throw new Error([state.message, state.description].filter(Boolean).join(": "));
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Device connected",
    variant: "success",
    details: { rack: rackKey, connected: true },
  });
  set(path.status, devStatus, { notifyOnChange: false, markTouched: false });
  return true;
};

const APIKeyField = () => {
  const name = Form.useFieldValue(path.name);
  return (
    <Secret.Field
      path={path.properties.apiKey}
      label="API key"
      defaultName={`${name || "Sift"} API key`}
      padHelpText={false}
      required
    />
  );
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
      if (deviceKey == null) capture("device_connected", { integration: INTEGRATION });
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION,
    name: "Sift",
    rack: rackKey,
  });
  return (
    <PlatformDevice.ConnectFrame className={CSS.B("sift-connect")}>
      <Modal.Header icon={<Icon.Logo.Sift />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large" justify="start">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections className={CSS.BE("sift-connect", "sections")}>
            <Form.Section title="Connection">
              <Form.TextField
                path={path.properties.uri}
                label="URI"
                padHelpText={false}
                required
                inputProps={URI_INPUT_PROPS}
              />
              <APIKeyField />
              <PlatformDevice.RackField integration={INTEGRATION} />
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

export const useConnectModal = PlatformDevice.createConnectModal(Connect, INTEGRATION);

const HEADER = ["Device", "Connect", "Sift"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Sift",
  "aria-label": "Name",
  autoFocus: true,
} as const;

const URI_INPUT_PROPS = { placeholder: "grpc-api.siftstack.com:443" } as const;
