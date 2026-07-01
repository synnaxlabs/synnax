// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Notifications } from "@/component/notifications";
import { versionOutdatedAdapter } from "@/service/cluster/notification";

export * from "@/service/cluster/Badges";
export * from "@/service/cluster/detectConnection";
export * from "@/service/cluster/link";
export * from "@/service/cluster/list";
export * from "@/service/cluster/palette";
export * from "@/service/cluster/useConnectModal";
export * from "@/service/cluster/useSyncClusterKey";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
