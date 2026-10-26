// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { NETWORK } from "@/feature/opcua/device/network";
import { MAKE } from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { Device as PlatformDevice } from "@/platform/device";

export const Select = () => {
  const openConnect = useConnectModal();
  return (
    <PlatformDevice.Select
      onConfigure={(deviceKey) => void openConnect({ deviceKey })}
      emptyContent="No OPC UA servers connected"
      label="OPC UA server"
      make={MAKE}
      icon={<Icon.Logo.OPCUA />}
      connect={{ title: NETWORK.title, open: () => openConnect() }}
    />
  );
};
