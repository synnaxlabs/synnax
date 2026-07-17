// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Notification } from "@/feature/version/Updater";
import { type Notifications } from "@/platform/notifications";

export * from "@/feature/version/Badge";
export * from "@/feature/version/Updater";
export * from "@/feature/version/useInfoModal";

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];
