// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { ZERO_SLAVE_PROPERTIES } from "@/feature/ethercat/device/types";
import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";

export const useConfigureModal = Modals.create<Device.ConfigureParams>(
  ({ deviceKey, close }) => (
    <Device.Configure
      deviceKey={deviceKey}
      close={close}
      icon={<Icon.Logo.EtherCAT />}
      initialProperties={ZERO_SLAVE_PROPERTIES}
    />
  ),
);
