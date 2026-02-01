// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device, Form } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
} from "@/hardware/ethercat/device/types";

export interface SelectSlaveProps {
  /** Path to the slave device field in the form. */
  path: string;
  /** Path to the channels array to get already-selected slaves. */
  channelsPath?: string;
}

export const SelectSlave = ({
  path,
  channelsPath = "config.channels",
}: SelectSlaveProps): ReactElement => {
  const channels = Form.useFieldValue<Array<{ device: string }>>(channelsPath) ?? [];

  const firstDeviceKey = useMemo(() => {
    const keys = channels.map((ch) => ch.device).filter(Boolean);
    return keys.length > 0 ? keys[0] : "";
  }, [channels]);

  const { data: firstDevice } = Device.useRetrieve({ key: firstDeviceKey });

  const selectedNetwork = useMemo(() => {
    if (!firstDevice) return "";
    return (firstDevice.properties as SlaveProperties | undefined)?.network ?? "";
  }, [firstDevice]);

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
            (selectedNetwork === "" ||
              (d.properties as SlaveProperties | undefined)?.network ===
                selectedNetwork)
          }
          emptyContent="No EtherCAT slaves discovered."
          grow
          variant={variant}
        />
      )}
    </Form.Field>
  );
};
