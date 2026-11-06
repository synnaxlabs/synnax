// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { useState } from "react";

import { useSlave, useUpdateSyncMode } from "@/feature/ethercat/device/queries";
import { type OpMode } from "@/feature/ethercat/device/types";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const AUTOMATIC = "automatic";

const formatMode = ({ name, assignActivate }: OpMode, i: number): string =>
  `${name === "" ? `Mode ${i + 1}` : name} (${assignActivate === 0 ? "SM" : "DC"} sync)`;

interface SyncModeParams {
  deviceKey: device.Key;
}

const SyncMode = ({ deviceKey, close }: Modals.ContentProps<SyncModeParams, void>) => {
  const slave = useSlave({ key: deviceKey });
  const { opModes, dcMode } = slave.properties;
  const [value, setValue] = useState(dcMode == null ? AUTOMATIC : dcMode.toString());
  const { update, variant } = useUpdateSyncMode({ afterSuccess: () => close() });
  const handleSave = () =>
    update({
      key: deviceKey,
      dcMode: value === AUTOMATIC ? undefined : Number(value),
    });
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Logo.EtherCAT />}>
        {[slave.name, "Synchronization mode"]}
      </Modal.Header>
      <Modal.Body align="stretch" gap="large">
        <Text.Text color={9}>
          The Driver sets this mode when the bus starts. To apply a change, stop every
          task on this network, then start them again.
        </Text.Text>
        <Select.Simple<string> value={value} onChange={setValue} resourceName="mode">
          <Select.Item itemKey={AUTOMATIC}>Automatic (SM sync when listed)</Select.Item>
          {opModes.map((mode, i) => (
            <Select.Item key={i.toString()} itemKey={i.toString()}>
              {formatMode(mode, i)}
            </Select.Item>
          ))}
        </Select.Simple>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={handleSave}
            variant="filled"
            trigger={Triggers.SAVE}
            triggerIndicator
            type="submit"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useSyncModeModal = Modals.create(SyncMode);
