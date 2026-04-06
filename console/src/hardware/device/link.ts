// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CONFIGURE_LAYOUTS, getMake } from "@/hardware/device/make";
import { type Link } from "@/link";

export const handleLink: Link.Handler = async ({ client, key, placeLayout }) => {
  const device = await client.devices.retrieve({ key });
  const make = getMake(device.make);
  if (make == null) return;
  placeLayout({ ...CONFIGURE_LAYOUTS[make], key: device.key, name: device.name });
};
