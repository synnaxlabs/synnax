// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { udp } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";

import {
  type Device,
  INTEGRATION_KEY,
  MAKE,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/udp/device/types";
import { Analytics } from "@/platform/analytics";
import { Device as PlatformDevice } from "@/platform/device";
import { Rack } from "@/platform/rack";

const path = deep.path<Device>();
const remotePath = path.properties.remote;
const shownRemote = deep.present(remotePath);

const INITIAL_VALUES: Device = {
  key: "",
  name: "",
  make: MAKE,
  model: MAKE,
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const ZERO_REMOTE: udp.Remote = { host: "", port: 0 };

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

/** Sets the location to the remote, or to the local address of a device with none. */
const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const { localAddress, localPort, remote } = get(path.properties).value;
  set(
    path.location,
    remote == null
      ? PlatformDevice.formatAddress(localAddress, localPort)
      : PlatformDevice.formatAddress(remote.host, remote.port),
  );
};

/** Turns the remote on and off. A device with no remote only receives. */
const RemoteFields = () => {
  const { set, remove } = Form.useContext();
  const value = Form.useFieldValue(remotePath, { optional: true });
  const handleChange = (on: boolean) => {
    if (on) set(remotePath, ZERO_REMOTE);
    else remove(remotePath);
  };
  return (
    <>
      <Input.Item label="Send to remote" padHelpText={false}>
        <Input.Switch value={value != null} onChange={handleChange} />
      </Input.Item>
      {value != null && (
        <>
          <Form.TextField
            path={shownRemote.host}
            label="Host"
            padHelpText={false}
            required
            inputProps={HOST_INPUT_PROPS}
          />
          <Form.NumericField
            path={shownRemote.port}
            label="Port"
            padHelpText={false}
            required
          />
        </>
      )}
    </>
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
    afterSave: ({ get }) => {
      if (deviceKey == null)
        capture("device_connected", { integration: INTEGRATION_KEY });
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = Rack.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "UDP",
    rack: rackKey,
  });
  return (
    <PlatformDevice.ConnectFrame>
      <Modal.Header icon={<Icon.Network />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large" justify="start">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections>
            <Form.Section title="Local">
              <Form.TextField
                path={path.properties.localAddress}
                label="Address"
                padHelpText={false}
                inputProps={LOCAL_ADDRESS_INPUT_PROPS}
              />
              <Form.NumericField
                path={path.properties.localPort}
                label="Port"
                padHelpText={false}
              />
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
            </Form.Section>
            <Form.Section title="Remote">
              <RemoteFields />
            </Form.Section>
          </Form.Sections>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onCancel={() => close()}
        onBack={onBack}
      />
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const HEADER = ["Device", "Connect", "UDP device"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Device name",
  "aria-label": "Name",
  autoFocus: true,
} as const;

const LOCAL_ADDRESS_INPUT_PROPS = { placeholder: "All interfaces" } as const;

const HOST_INPUT_PROPS = { placeholder: "192.168.1.20" } as const;
