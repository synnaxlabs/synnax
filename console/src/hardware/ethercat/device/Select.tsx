// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { CONFIGURE_LAYOUT } from "@/hardware/ethercat/device/Configure";
import { MAKE, NETWORK_MODEL } from "@/hardware/ethercat/device/types";

export const Select = () => (
  <Common.Device.Select
    configureLayout={CONFIGURE_LAYOUT}
    emptyContent="No EtherCAT networks discovered."
    label="EtherCAT Network"
    make={MAKE}
    model={NETWORK_MODEL}
  />
);

export interface SelectNetworkProps {
  path?: string;
}

export const SelectNetwork = ({ path = "config.device" }: SelectNetworkProps) => (
  <Common.Device.Select
    configureLayout={CONFIGURE_LAYOUT}
    emptyContent="No EtherCAT networks discovered."
    label="EtherCAT Network"
    make={MAKE}
    model={NETWORK_MODEL}
    path={path}
  />
);
