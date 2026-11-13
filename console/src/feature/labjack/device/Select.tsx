// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HARDWARE } from "@/feature/labjack/device/hardware";
import { MAKE } from "@/feature/labjack/device/types";
import { Device } from "@/platform/device";

const EMPTY_CONTENT = "No LabJack devices connected";

const useConnect = (): Device.SelectConnect => {
  const find = Device.useFindModal();
  return { title: HARDWARE.title, open: () => find({ hardware: HARDWARE }) };
};

export const useNoneSelected = (): Device.NoneSelectedParams => ({
  make: MAKE,
  emptyContent: EMPTY_CONTENT,
  connect: useConnect(),
});

export const Select = () => {
  const configure = Device.useConfigureModal();
  const connect = useConnect();
  return (
    <Device.Select
      onConfigure={(deviceKey) => configure({ deviceKey, hardware: HARDWARE })}
      emptyContent={EMPTY_CONTENT}
      make={MAKE}
      icon={HARDWARE.icon}
      locationFormat={HARDWARE.location}
      connect={connect}
    />
  );
};
