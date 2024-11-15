// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Embedded } from "@/cluster/embedded";
import { versionOutdatedAdapter } from "@/cluster/notification";
import { Remote } from "@/cluster/remote";
import { type Layout } from "@/layout";
import { type NotificationAdapter } from "@/notifications/Notifications";

export { Boundary } from "@/cluster/remote/Boundary";
export * from "@/cluster/selectors";
export * from "@/cluster/slice";
export * from "@/cluster/testConnection";
export * from "@/cluster/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Remote.LAYOUTS,
  ...Embedded.LAYOUTS,
};

export const NOTIFICATION_ADAPTERS: NotificationAdapter[] = [versionOutdatedAdapter];
