// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { versionOutdatedAdapter } from "@/node/notification";
import { type Notifications } from "@/notifications";

export * from "@/node/Badges";
export * from "@/node/CopyLinkToolbarButton";
export * from "@/node/detectConnection";
export * from "@/node/list";
export * from "@/node/selectors";
export * from "@/node/slice";
export * from "@/node/useConnectModal";
export * from "@/node/useCopyLinkToClipboard";
export * from "@/node/useSyncClusterKey";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
