// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { ZERO_PROPERTIES } from "@/hardware/ni/device/types";
import { type Layout } from "@/layout";

export const CONFIGURE_LAYOUT_TYPE = "configure_NI";

export const CONFIGURE_LAYOUT: Layout.BaseState = {
  ...Common.Device.CONFIGURE_LAYOUT,
  name: "Device.Configure",
  icon: "Logo.NI",
  type: CONFIGURE_LAYOUT_TYPE,
  key: CONFIGURE_LAYOUT_TYPE,
};

export const Configure: Layout.Renderer = (props) => (
  <Common.Device.Configure {...props} initialProperties={ZERO_PROPERTIES} />
);
