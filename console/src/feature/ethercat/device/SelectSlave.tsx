// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type ethercat } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";

import { HARDWARE } from "@/feature/ethercat/device/hardware";
import { useCommonNetwork } from "@/feature/ethercat/device/queries";
import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";
import { Device } from "@/platform/device";

export interface SelectSlaveProps {
  /** Path to the slave device field in the form. */
  path: string;
  /** Path to the channels array to get already-selected slaves. */
  channelsPath?: string;
}

const filterByNetwork = (d: device.Device, network: string) =>
  network === "" || (d.properties as SlaveProperties | undefined)?.network === network;

export const SelectSlave = ({
  path,
  channelsPath = "config.channels",
}: SelectSlaveProps): ReactElement => {
  const channels =
    Form.useFieldValue<(ethercat.ReadChannel | ethercat.WriteChannel)[]>(
      channelsPath,
    ) ?? [];
  const network = useCommonNetwork(channels);
  const configure = Device.useConfigureModal();
  const find = Device.useFindModal();
  const filter = (d: device.Device) => filterByNetwork(d, network);
  return (
    <Device.Select
      path={path}
      label="Slave device"
      onConfigure={(deviceKey) => configure({ deviceKey, hardware: HARDWARE })}
      emptyContent="No EtherCAT slaves discovered"
      make={MAKE}
      model={SLAVE_MODEL}
      filter={filter}
      connect={{
        title: HARDWARE.title,
        open: () => find({ hardware: HARDWARE, filter }),
      }}
    />
  );
};
