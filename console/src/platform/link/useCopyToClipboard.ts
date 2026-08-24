// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Clipboard } from "@/platform/clipboard";
import { PREFIX } from "@/platform/link/types";

export interface CopyToClipboardParams {
  /** The cluster the link opens. A Core caches it on its first connection. */
  clusterKey?: string;
  name: string;
  ontologyID?: ontology.ID;
}

export interface CopyToClipboard {
  (params: CopyToClipboardParams): void;
}

export const useCopyToClipboard = (): CopyToClipboard => {
  const copy = Clipboard.useCopy();
  const addStatus = Status.useAdder();
  return useCallback(
    ({ clusterKey, name, ontologyID }) => {
      if (clusterKey == null) {
        addStatus({
          variant: "error",
          message: `Failed to copy link to ${name}`,
          description: "Connect to the Core to get a link to its cluster.",
        });
        return;
      }
      let url = `${PREFIX}${clusterKey}`;
      if (ontologyID != null) url += `/${ontologyID.type}/${ontologyID.key}`;
      return copy(url, `link to ${name}`);
    },
    [copy, addStatus],
  );
};
