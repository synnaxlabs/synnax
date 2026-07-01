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

export * from "@/feature/cluster/Badges";
export * from "@/feature/cluster/detectConnection";
export * from "@/feature/cluster/link";
export * from "@/feature/cluster/list";
export * from "@/feature/cluster/palette";
export * from "@/feature/cluster/useConnectModal";
export * from "@/feature/cluster/useSyncClusterKey";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
