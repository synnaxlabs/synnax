// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Cluster } from "@/feature/cluster";
import { Device } from "@/feature/device";
import { type Notifications } from "@/platform/notifications";
import { Version } from "@/platform/version";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [
  ...Cluster.NOTIFICATION_ADAPTERS,
  ...Device.NOTIFICATION_ADAPTERS,
  ...Version.NOTIFICATION_ADAPTERS,
];
