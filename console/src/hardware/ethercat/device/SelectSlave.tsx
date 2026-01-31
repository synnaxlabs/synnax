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
import { type ReactElement } from "react";

import { MAKE, SLAVE_MODEL } from "@/hardware/ethercat/device/types";

export interface SelectSlaveProps {
  /** Path to the slave device field in the form. */
  path: string;
  /** Path to the network device field to filter slaves by. */
  networkPath?: string;
}

export const SelectSlave = ({
  path,
  networkPath = "config.device",
}: SelectSlaveProps): ReactElement => {
  const networkDevice = Form.useFieldValue<device.Key>(networkPath);
  return (
    <Form.Field<string> path={path} label="Slave Device" grow>
      {({ value, onChange, variant }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialQuery={{ makes: [MAKE] }}
          filter={(d) =>
            d.make === MAKE &&
            d.model === SLAVE_MODEL &&
            (networkDevice === "" || networkDevice.endsWith(d.location))
          }
          emptyContent="No EtherCAT slaves discovered."
          grow
          variant={variant}
        />
      )}
    </Form.Field>
  );
};
