// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Device } from "@/component/device";
import { useCommonNetwork } from "@/service/ethercat/device/queries";
import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
} from "@/service/ethercat/device/types";
import { useConfigureModal } from "@/service/ethercat/device/useConfigureModal";
import { type Channel } from "@/service/ethercat/task/types";

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
  const channels = Form.useFieldValue<Channel[]>(channelsPath) ?? [];
  const network = useCommonNetwork(channels);
  const configure = useConfigureModal();
  const filter = useCallback(
    (d: device.Device) => filterByNetwork(d, network),
    [network],
  );
  return (
    <Device.Select
      path={path}
      label="Slave Device"
      onConfigure={(deviceKey) => configure({ deviceKey })}
      emptyContent="No EtherCAT slaves discovered."
      make={MAKE}
      model={SLAVE_MODEL}
      filter={filter}
    />
  );
};
