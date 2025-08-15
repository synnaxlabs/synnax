// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { CONFIGURE_LAYOUT } from "@/hardware/ni/device/Configure";
import { MAKE } from "@/hardware/ni/device/types";

export interface SelectProps
  extends Omit<
    Common.Device.SelectProps,
    "configureLayout" | "emptyContent" | "make"
  > {}

export const Select = (props: SelectProps) => (
  <Common.Device.Select
    {...props}
    configureLayout={CONFIGURE_LAYOUT}
    emptyContent="No NI devices connected."
    make={MAKE}
    icon={<Icon.Logo.NI />}
  />
);
