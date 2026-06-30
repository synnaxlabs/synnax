// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Notifications } from "@/notifications";
import { notificationAdapter } from "@/service/version/Updater";

export * from "@/service/version/Badge";
export * from "@/service/version/Info";
export * from "@/service/version/Updater";
export * from "@/service/version/use";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [notificationAdapter];
