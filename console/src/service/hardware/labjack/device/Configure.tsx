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
import { ZERO_PROPERTIES } from "@/hardware/labjack/device/types";
import { Modals } from "@/service/modals";

export const useConfigureModal = Modals.create<Common.Device.ConfigureParams>(
  ({ deviceKey, close }) => (
    <Common.Device.Configure
      deviceKey={deviceKey}
      close={close}
      icon={<Icon.Logo.LabJack />}
      initialProperties={ZERO_PROPERTIES}
    />
  ),
);
