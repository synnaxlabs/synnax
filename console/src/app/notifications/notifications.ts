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
import { Notifications as Base } from "@/platform/notifications";
import { Version } from "@/platform/version";

const suppressRoutine = (prefix: string): Base.Notification =>
  Base.createSuppressed(
    (status) =>
      (status.variant === "success" || status.variant === "loading") &&
      status.key.startsWith(prefix),
  );

export const NOTIFICATIONS: Base.Notification[] = [
  suppressRoutine("rack"),
  suppressRoutine("device"),
  suppressRoutine("task"),
  ...Cluster.NOTIFICATIONS,
  ...Device.NOTIFICATIONS,
  ...Version.NOTIFICATIONS,
];
