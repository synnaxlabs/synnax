// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NETWORK } from "@/feature/opcua/device/network";
import { MAKE } from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { Device as PlatformDevice } from "@/platform/device";

const EMPTY_CONTENT = "No OPC UA servers connected";

const useConnect = (): PlatformDevice.SelectConnect => {
  const openConnect = useConnectModal();
  return { title: NETWORK.title, open: () => openConnect() };
};

export const useNoneSelected = (): PlatformDevice.NoneSelectedParams => ({
  make: MAKE,
  emptyContent: EMPTY_CONTENT,
  connect: useConnect(),
});

export const Select = () => {
  const openConnect = useConnectModal();
  const connect = useConnect();
  return (
    <PlatformDevice.Select
      onConfigure={(deviceKey) => void openConnect({ deviceKey })}
      emptyContent={EMPTY_CONTENT}
      label="OPC UA server"
      make={MAKE}
      icon={NETWORK.icon}
      locationFormat={NETWORK.location}
      connect={connect}
    />
  );
};
