// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";

import { Clipboard } from "@/platform/clipboard";
import { PREFIX } from "@/platform/link/types";

export interface CopyToClipboardParams {
  /** The cluster the link opens. A Core caches it on its first connection. */
  clusterKey?: string;
  name: string;
  resourceID?: resource.ID;
}

export interface CopyToClipboard {
  (params: CopyToClipboardParams): void;
}

export const useCopyToClipboard = (): CopyToClipboard => {
  const copy = Clipboard.useCopy();
  const addStatus = Status.useAdder();
  return ({ clusterKey, name, resourceID }) => {
    if (clusterKey == null) {
      addStatus({
        variant: "error",
        name,
        message: "Failed to copy link",
        description: "Connect to the Core to get a link to its cluster.",
      });
      return;
    }
    let url = `${PREFIX}${clusterKey}`;
    if (resourceID != null) url += `/${resourceID.type}/${resourceID.key}`;
    return copy(url, "link", name);
  };
};
