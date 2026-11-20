// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NETWORK } from "@/feature/kafka/device/network";
import { MAKE } from "@/feature/kafka/device/types";
import { useConnectModal } from "@/feature/kafka/device/useConnectModal";
import { Device } from "@/platform/device";

const EMPTY_CONTENT = "No Kafka clusters connected";

const useConnect = (): Device.SelectConnect => {
  const openConnect = useConnectModal();
  return { title: NETWORK.title, open: () => openConnect() };
};

export const useNoneSelected = (): Device.NoneSelectedParams => ({
  make: MAKE,
  emptyContent: EMPTY_CONTENT,
  connect: useConnect(),
});

export const Select = () => {
  const openConnect = useConnectModal();
  const connect = useConnect();
  return (
    <Device.Select
      onConfigure={(deviceKey) => void openConnect({ deviceKey })}
      emptyContent={EMPTY_CONTENT}
      label="Kafka cluster"
      make={MAKE}
      icon={NETWORK.icon}
      locationFormat={NETWORK.location}
      connect={connect}
    />
  );
};
