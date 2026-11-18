// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION, MAKE } from "@/feature/sift/device/types";
import { Connect } from "@/feature/sift/device/useConnectModal";
import { SCAN_TYPE } from "@/feature/sift/task/types";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION,
  name: "Sift",
  title: "Sift",
  description: "Send live and recorded data to Sift for review.",
  icon: <Icon.Logo.Sift />,
  make: MAKE,
  scanType: SCAN_TYPE,
  location: { shape: "address", hiddenSchemes: ["https://", "http://"] },
  Connect,
};
