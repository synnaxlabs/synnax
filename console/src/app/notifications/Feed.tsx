// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel } from "@/feature/channel";
import { Core } from "@/feature/core";
import { Device } from "@/feature/device";
import { Rack } from "@/feature/rack";
import { Task } from "@/feature/task";
import { Notifications } from "@/platform/notifications";
import { Version } from "@/platform/version";

const NOTIFICATIONS: Notifications.Notification[] = [
  ...(DESKTOP ? [] : Core.NOTIFICATIONS),
  ...Device.NOTIFICATIONS,
  ...Version.NOTIFICATIONS,
];

const RULES: Notifications.Rules = {
  channel: Channel.NOTIFICATION_RULE,
  device: Device.NOTIFICATION_RULE,
  rack: Rack.NOTIFICATION_RULE,
  task: Task.createNotificationRule(Device.INTEGRATIONS),
};

export const Feed = () => <Notifications.Feed notifications={NOTIFICATIONS} />;

/** Notifies the status writes that the Core streams by the rules of each feature. */
export const useForward = (): void => Notifications.useForward(RULES);
