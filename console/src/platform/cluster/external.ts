// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { versionOutdatedAdapter } from "@/platform/cluster/notification";
import { type Notifications } from "@/platform/notifications";

export * from "@/platform/cluster/Badges";
export * from "@/platform/cluster/CopyLinkToolbarButton";
export * from "@/platform/cluster/detectConnection";
export * from "@/platform/cluster/link";
export * from "@/platform/cluster/list";
export * from "@/platform/cluster/palette";
export * from "@/platform/cluster/useConnectModal";
export * from "@/platform/cluster/useCopyLinkToClipboard";
export * from "@/platform/cluster/useSyncClusterKey";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
