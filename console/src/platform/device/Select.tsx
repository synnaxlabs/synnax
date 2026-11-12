// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/Select.css";

import { device } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Access, Device } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Button } from "@/platform/button";
import { CSS } from "@/platform/css";
import { useHandleChange } from "@/platform/device/useHandleChange";

const root = deep.path<{ config: { device: device.Key } }>();

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
  path?: deep.Target<device.Key>;
  icon: Icon.ReactElement;
  /** How the location of each listed device reads. */
  locationFormat?: Device.LocationFormat;
  /** Adds an action below the list that connects a device and selects it. */
  connect?: SelectConnect;
  /** Runs after the task form handles a change of device. */
  onChange?: (key: device.Key, ctx: Form.ContextValue) => void;
}

export const Select = ({
  onConfigure,
  emptyContent = "No devices connected",
  filter: filterProp,
  label = "Device",
  make,
  model,
  path = root.config.device,
  icon,
  locationFormat,
  connect,
  onChange,
}: SelectProps) => {
  const canConnect = Access.useCreateGranted(device.TYPE_RESOURCE_ID);
  const handleChange = useHandleChange(onConfigure);
  const [visible, setVisible] = useState(false);
  Form.useFocusHandler(path, () => setVisible(true));
  const filter = (d: device.Device) =>
    d.make === make &&
    (model == null || d.model === model) &&
    (filterProp?.(d) ?? true);
  return (
    <Form.Field
      className={CSS.B("device-select")}
      grow
      label={label}
      onChange={(key, ctx) => {
        handleChange(key, ctx);
        onChange?.(key, ctx);
      }}
      path={path}
    >
      {({ value, onChange, preview }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          query={{ where: { make: { eq: make } } }}
          filter={filter}
          emptyContent={emptyContent}
          grow
          icon={icon}
          locationFormat={locationFormat}
          preview={preview}
          visible={visible}
          onVisibleChange={setVisible}
          footer={
            connect != null &&
            canConnect && <ConnectButton {...connect} onChange={onChange} />
          }
        />
      )}
    </Form.Field>
  );
};
