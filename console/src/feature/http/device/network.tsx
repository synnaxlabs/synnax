// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { SCAN_TYPE } from "@/feature/http/device/scan";
import { INTEGRATION, MAKE } from "@/feature/http/device/types";
import { Connect } from "@/feature/http/device/useConnectModal";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION,
  name: "HTTP",
  title: "HTTP server",
  description: "Poll and send JSON to an HTTP API.",
  icon: <Icon.Logo.HTTP />,
  make: MAKE,
  scanType: SCAN_TYPE,
  Connect,
};
