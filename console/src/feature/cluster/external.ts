// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { versionOutdatedAdapter } from "@/feature/cluster/notification";
import { type Notifications } from "@/platform/notifications";

export * from "@/feature/cluster/commands";
export * from "@/feature/cluster/ConnectionBadge";
export * from "@/feature/cluster/link";
export * from "@/platform/cluster/external";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [versionOutdatedAdapter];
