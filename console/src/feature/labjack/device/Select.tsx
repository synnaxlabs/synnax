// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { HARDWARE } from "@/feature/labjack/device/hardware";
import { MAKE } from "@/feature/labjack/device/types";
import { Device } from "@/platform/device";

export const Select = () => {
  const configure = Device.useConfigureModal();
  const find = Device.useFindModal();
  return (
    <Device.Select
      onConfigure={(deviceKey) => configure({ deviceKey, hardware: HARDWARE })}
      emptyContent="No LabJack devices connected"
      make={MAKE}
      icon={<Icon.Logo.LabJack />}
      connect={{ title: HARDWARE.title, open: () => find({ hardware: HARDWARE }) }}
    />
  );
};
