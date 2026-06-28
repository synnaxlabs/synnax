// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { useConfigure } from "@/hardware/labjack/device/Configure";
import { MAKE } from "@/hardware/labjack/device/types";

export const Select = () => {
  const configure = useConfigure();
  return (
    <Common.Device.Select
      onConfigure={(deviceKey) => configure({ deviceKey })}
      emptyContent="No LabJack devices connected."
      make={MAKE}
      icon={<Icon.Logo.LabJack />}
    />
  );
};
