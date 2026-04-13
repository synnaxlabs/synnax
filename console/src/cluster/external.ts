// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Connect, CONNECT_LAYOUT_TYPE } from "@/cluster/Connect";
import { versionOutdatedAdapter } from "@/cluster/notification";
import { type Layout } from "@/layout";
import { type Notifications } from "@/notifications";

export * from "@/cluster/Badges";
export * from "@/cluster/Connect";
export * from "@/cluster/CopyLinkToolbarButton";
export * from "@/cluster/detectConnection";
export * from "@/cluster/list";
export * from "@/cluster/selectors";
export * from "@/cluster/slice";
export * from "@/cluster/useCopyLinkToClipboard";
export * from "@/cluster/useSyncClusterKey";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONNECT_LAYOUT_TYPE]: Connect,
};

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
