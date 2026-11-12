// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { HARDWARE } from "@/feature/ethercat/device/hardware";
import { useCommonNetwork } from "@/feature/ethercat/device/queries";
import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";
import { type Channel } from "@/feature/ethercat/task/types";
import { Device } from "@/platform/device";

const root = deep.path<{ config: { channels: Channel[] } }>();

export interface SelectSlaveProps extends Pick<Device.SelectProps, "onChange"> {
  /** Path to the slave device field in the form. */
  path: deep.Target<device.Key>;
}

const filterByNetwork = (d: device.Device, network: string) =>
  network === "" || (d.properties as SlaveProperties | undefined)?.network === network;

export const SelectSlave = ({ path, onChange }: SelectSlaveProps): ReactElement => {
  const channels = Form.useFieldValue(root.config.channels) ?? [];
  const network = useCommonNetwork(channels);
  const configure = Device.useConfigureModal();
  const find = Device.useFindModal();
  const filter = (d: device.Device) => filterByNetwork(d, network);
  return (
    <Device.Select
      path={path}
      label="Device"
      onConfigure={(deviceKey) => configure({ deviceKey, hardware: HARDWARE })}
      emptyContent="No EtherCAT devices discovered"
      make={MAKE}
      model={SLAVE_MODEL}
      icon={HARDWARE.icon}
      locationFormat={HARDWARE.location}
      filter={filter}
      onChange={onChange}
      connect={{
        title: HARDWARE.title,
        open: () => find({ hardware: HARDWARE, filter }),
      }}
    />
  );
};
