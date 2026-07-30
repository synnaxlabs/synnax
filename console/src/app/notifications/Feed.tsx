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
import { Rack } from "@/feature/rack";
import { Task } from "@/feature/task";
import { Version } from "@/platform/version";
import { Notifications } from "@/platform/notifications";

const NOTIFICATIONS: Notifications.Notification[] = [
  ...Cluster.NOTIFICATIONS,
  ...Device.NOTIFICATIONS,
  ...Rack.NOTIFICATIONS,
  ...Task.NOTIFICATIONS,
  ...Version.NOTIFICATIONS,
];

export const Feed = () => <Notifications.Feed notifications={NOTIFICATIONS} />;
