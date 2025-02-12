// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";

import { Cluster } from "@/cluster";
import { PREFIX } from "@/link/types";

export interface CopyToClipboardProps {
  clusterKey?: string;
  name?: string;
  ontologyID?: ontology.IDPayload;
}

export const useCopyToClipboard = (): ((props: CopyToClipboardProps) => void) => {
  const activeClusterKey = Cluster.useSelectActiveKey();
  const addStatus = Status.useAggregator();
  return ({ ontologyID, name, clusterKey }) => {
    let url = PREFIX;
    const key = clusterKey ?? activeClusterKey;
    const linkMessage = name == null ? "" : `to ${name}`;
    if (key == null) {
      addStatus({
        variant: "error",
        message: `Failed to copy link ${linkMessage} to clipboard`,
        description: "No active cluster found",
      });
      return;
    }
    url += key;
    if (ontologyID != undefined) url += `/${ontologyID.type}/${ontologyID.key}`;
    navigator.clipboard.writeText(url).then(
      () =>
        addStatus({
          variant: "success",
          message: `Link ${linkMessage} copied to clipboard.`,
        }),
      () => {
        addStatus({
          variant: "error",
          message: `Failed to copy link ${linkMessage} to clipboard.`,
        });
      },
    );
  };
};
