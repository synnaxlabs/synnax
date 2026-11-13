// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NETWORK } from "@/feature/sift/device/network";
import { MAKE } from "@/feature/sift/device/types";
import { useConnectModal } from "@/feature/sift/device/useConnectModal";
import { Device } from "@/platform/device";

export interface SelectProps extends Pick<Device.SelectProps, "onChange"> {}

export const Select = ({ onChange }: SelectProps) => {
  const openConnect = useConnectModal();
  return (
    <Device.Select
      onChange={onChange}
      onConfigure={(deviceKey) => void openConnect({ deviceKey })}
      emptyContent="No Sift connections"
      label="Sift connection"
      make={MAKE}
      icon={NETWORK.icon}
      locationFormat={NETWORK.location}
      connect={{ title: NETWORK.title, open: () => openConnect() }}
    />
  );
};
