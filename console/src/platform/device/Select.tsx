// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Device, Synnax } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/platform/button";

export interface SelectConnect {
  /** What the action connects, such as "OPC UA server". */
  title: string;
  /** Opens the connect flow. Resolves with the device to select, or null. */
  open: () => Promise<device.Key | null>;
}

interface ConnectButtonProps extends SelectConnect {
  onChange: (key: device.Key) => void;
}

const ConnectButton = ({ title, open, onChange }: ConnectButtonProps): ReactElement => {
  const { close } = Dialog.useContext();
  const handleClick = async () => {
    close();
    const key = await open();
    if (key != null) onChange(key);
  };
  return (
    <Button.CreateListItem size="small" onClick={() => void handleClick()}>
      Connect {title}
    </Button.CreateListItem>
  );
};

export interface SelectProps extends Pick<Device.SelectSingleProps, "filter"> {
  onConfigure: (deviceKey: device.Key) => void;
  emptyContent?: string;
  label?: string;
  make: string;
  model?: string;
  path?: string;
  icon?: Icon.ReactElement;
  /** Adds an action below the list that connects a device and selects it. */
  connect?: SelectConnect;
}

export const Select = ({
  onConfigure,
  emptyContent = "No devices connected",
  filter: filterProp,
  label = "Device",
  make,
  model,
  path = "config.device",
  icon,
  connect,
}: SelectProps) => {
  const client = Synnax.use();
  const canConnect = Access.useCreateGranted(device.TYPE_ONTOLOGY_ID);
  const handleError = Status.useErrorHandler();
  const handleDeviceChange = (key: device.Key, { set }: Form.ContextValue) => {
    if (client == null || primitive.isZero(key)) return;
    handleError(async () => {
      const { configured, rack } = await client.devices.retrieve({ key });
      set("rack", rack);
      if (configured) return;
      onConfigure(key);
    }, "Failed to retrieve device");
  };
  const filter = (d: device.Device) =>
    d.make === make &&
    (model == null || d.model === model) &&
    (filterProp?.(d) ?? true);
  return (
    <Form.Field<string> grow label={label} onChange={handleDeviceChange} path={path}>
      {({ value, onChange, preview }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialQuery={{ where: { make: { eq: make } } }}
          filter={filter}
          emptyContent={emptyContent}
          grow
          icon={icon}
          preview={preview}
          footer={
            connect != null &&
            canConnect && <ConnectButton {...connect} onChange={onChange} />
          }
        />
      )}
    </Form.Field>
  );
};
