// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/pluto";

import { MAKE } from "@/feature/modbus/device/types";
import { useConnectModal } from "@/feature/modbus/device/useConnectModal";
import { Device } from "@/platform/device";
import { Empty } from "@/platform/empty";

const EmptyContent = () => {
  const connect = useConnectModal();
  const { close: closeDialog } = Dialog.useContext();
  return (
    <Empty.Action
      message="No Modbus servers connected."
      action="Connect a new server"
      onClick={() => {
        connect();
        closeDialog();
      }}
    />
  );
};

export const Select = () => {
  const connect = useConnectModal();
  return (
    <Device.Select
      onConfigure={(deviceKey) => connect({ deviceKey })}
      emptyContent={<EmptyContent />}
      label="Modbus Server"
      make={MAKE}
    />
  );
};
