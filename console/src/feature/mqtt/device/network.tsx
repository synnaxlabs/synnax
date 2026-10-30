// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION } from "@/feature/mqtt/device/types";
import { Connect } from "@/feature/mqtt/device/useConnectModal";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION,
  name: "MQTT",
  title: "MQTT broker",
  description: "Subscribe and publish to topics on an MQTT broker, with Sparkplug B.",
  icon: <Icon.Logo.MQTT />,
  Connect,
};
