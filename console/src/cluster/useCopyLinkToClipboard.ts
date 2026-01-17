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

import { useSelectActiveKey } from "@/cluster/selectors";
import { Link } from "@/link";

export interface CopyLinkToClipboardArgs {
  name: string;
  ontologyID: ontology.ID;
}

export interface CopyLinkToClipboard {
  (args: CopyLinkToClipboardArgs): void;
}

export const useCopyLinkToClipboard = (): CopyLinkToClipboard => {
  const copyLink = Link.useCopyToClipboard();
  const clusterKey = useSelectActiveKey();
  const addStatus = Status.useAdder();
  return useCallback(
    (args) => {
      if (clusterKey == null) {
        addStatus({
          variant: "error",
          message: `Failed to copy link to ${args.name}`,
          description: "No active cluster is found",
        });
        return;
      }
      return copyLink({ ...args, clusterKey });
    },
    [copyLink, clusterKey, addStatus],
  );
};
