// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { type Notifications } from "@/notifications";
import { Info, INFO_LAYOUT_TYPE } from "@/version/Info";
import { notificationAdapter } from "@/version/Updater";

export * from "@/version/Badge";
export * from "@/version/Info";
export * from "@/version/selectors";
export * from "@/version/slice";
export * from "@/version/tauriVersion";
export * from "@/version/Updater";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [INFO_LAYOUT_TYPE]: Info,
};

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [notificationAdapter];
