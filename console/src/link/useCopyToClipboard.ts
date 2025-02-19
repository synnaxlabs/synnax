// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { useCallback } from "react";

import { useCopyToClipboard as useCopy } from "@/hooks/useCopyToClipboard";
import { PREFIX } from "@/link/types";

export interface CopyToClipboardArgs {
  clusterKey: string;
  name: string;
  ontologyID?: ontology.IDPayload;
}

export const useCopyToClipboard = (): ((args: CopyToClipboardArgs) => void) => {
  const copy = useCopy();
  return useCallback(
    ({ clusterKey, name, ontologyID }) => {
      let url = `${PREFIX}${clusterKey}`;
      if (ontologyID != null) url += `/${ontologyID.type}/${ontologyID.key}`;
      return copy(url, `link to ${name}`);
    },
    [copy],
  );
};
