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
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import type { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Device } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import {
  type RetrieveSelected,
  useHandleChange,
} from "@/platform/device/useHandleChange";

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

/** The props that a select gives the list of devices it selects from. */
export interface SelectListProps extends Pick<
  Device.SelectSingleProps,
  | "value"
  | "onChange"
  | "preview"
  | "visible"
  | "onVisibleChange"
  | "footer"
  | "grow"
  | "emptyContent"
  | "resourceName"
  | "icon"
> {}

/** Where a select lists and reads devices that live outside the generic table. */
export interface SelectSource {
  /** Lists the devices to select from. */
  List: ComponentType<SelectListProps>;
  /** Reads the rack and configured state of the selected device. */
  retrieve: RetrieveSelected;
}

export interface SelectProps extends Pick<Device.SelectSingleProps, "filter"> {
  onConfigure: (deviceKey: device.Key) => void;
  emptyContent?: string;
  label?: string;
  /** What the select calls a device, such as "connection". */
  resourceName?: string;
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
  /** Lists and reads the devices from another table. Absent for the generic table. */
  source?: SelectSource;
}

export const Select = ({
  onConfigure,
  emptyContent = "No devices connected",
  filter: filterProp,
  label = "Device",
  resourceName,
  make,
  model,
  path = root.config.device,
  icon,
  locationFormat,
  connect,
  onChange,
  source,
}: SelectProps) => {
  const canConnect = Access.useCreateGranted(device.TYPE_RESOURCE_ID);
  const handleChange = useHandleChange(onConfigure, source?.retrieve);
  const [visible, setVisible] = useState(false);
  Form.useFocusHandler(path, () => setVisible(true));
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
      {({ value, onChange, preview }) => {
        const footer = connect != null && canConnect && (
          <ConnectButton {...connect} onChange={onChange} />
        );
        const shared = {
          value,
          onChange,
          preview,
          visible,
          onVisibleChange: setVisible,
          footer,
          grow: true,
          emptyContent,
          resourceName,
          icon,
        };
        if (source != null) return <source.List {...shared} />;
        return (
          <Device.SelectSingle
            {...shared}
            query={{
              where: {
                make: { eq: make },
                ...(model != null && { model: { eq: model } }),
              },
            }}
            filter={filterProp}
            locationFormat={locationFormat}
          />
        );
      }}
    </Form.Field>
  );
};
