// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NETWORK } from "@/feature/modbus/device/network";
import { MAKE } from "@/feature/modbus/device/types";
import { useConnectModal } from "@/feature/modbus/device/useConnectModal";
import { Device } from "@/platform/device";

export const Select = () => {
  const openConnect = useConnectModal();
  return (
    <Device.Select
      onConfigure={(deviceKey) => void openConnect({ deviceKey })}
      emptyContent="No Modbus servers connected"
      label="Modbus server"
      make={MAKE}
      connect={{ title: NETWORK.title, open: () => openConnect() }}
    />
  );
};
