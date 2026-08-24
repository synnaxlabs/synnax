// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { useCallback } from "react";

import { Link } from "@/platform/link";
import { Session } from "@/session";

export interface CopyLinkToClipboardParams {
  name: string;
  ontologyID: ontology.ID;
}

export interface CopyLinkToClipboard {
  (params: CopyLinkToClipboardParams): void;
}

export const useCopyLinkToClipboard = (): CopyLinkToClipboard => {
  const copyLink = Link.useCopyToClipboard();
  const clusterKey = Session.Core.useSelectClusterKey();
  return useCallback(
    (params) => copyLink({ ...params, clusterKey }),
    [copyLink, clusterKey],
  );
};
