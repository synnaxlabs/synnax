// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Form } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useCommonNetwork } from "@/hardware/ethercat/device/queries";
import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
} from "@/hardware/ethercat/device/types";
import { type Channel } from "@/hardware/ethercat/task/types";

export interface SelectSlaveProps {
  /** Path to the slave device field in the form. */
  path: string;
  /** Path to the channels array to get already-selected slaves. */
  channelsPath?: string;
}

const filter = (d: device.Device, network: string) =>
  d.make === MAKE &&
  d.model === SLAVE_MODEL &&
  (network === "" ||
    (d.properties as SlaveProperties | undefined)?.network === network);

export const SelectSlave = ({
  path,
  channelsPath = "config.channels",
}: SelectSlaveProps): ReactElement => {
  const channels = Form.useFieldValue<Array<Channel>>(channelsPath) ?? [];
  const network = useCommonNetwork(channels);
  const onFilter = useCallback((d: device.Device) => filter(d, network), [network]);
  return (
    <Form.Field<string> path={path} label="Slave Device" grow>
      {({ value, onChange, variant }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialQuery={{ makes: [MAKE] }}
          filter={onFilter}
          emptyContent="No EtherCAT slaves discovered."
          grow
          variant={variant}
        />
      )}
    </Form.Field>
  );
};
