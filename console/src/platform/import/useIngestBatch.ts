// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type panel } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";

/** Maps the resources ingest created to the tabs that open them. */
const resourceTabs = (ids: (ontology.ID | void)[]): panel.NewTab[] =>
  ids.filter((id) => id != null).map((resource) => ({ variant: "resource", resource }));

export interface IngestBatchParams<I extends { name: string }> {
  /** The items to ingest. An item's name identifies it when its ingest fails. */
  items: I[];
  /** Creates the resource the item describes, or nothing when it opens no tab. TODO:
   * should probably NOT return `void`. */
  ingest: (item: I) => Promise<void | ontology.ID>;
  /** Places the tabs in a specific leaf instead of beside the current one. */
  placement?: Panel.Placement;
}

export interface IngestBatch {
  <I extends { name: string }>(params: IngestBatchParams<I>): Promise<void>;
}

/**
 * Returns a batch ingester: it ingests every item concurrently, then opens the
 * resources they created as one batch of tabs. An item that fails is reported on its
 * own and leaves the rest of the batch running.
 */
export const useIngestBatch = (): IngestBatch => {
  const handleError = Status.useErrorHandler();
  const openTabs = Panel.useOpenTabs();
  return useCallback(
    async <I extends { name: string }>({
      items,
      ingest,
      placement,
    }: IngestBatchParams<I>): Promise<void> => {
      const ids = await Promise.all(
        items.map(async (item) => {
          try {
            return await ingest(item);
          } catch (e) {
            handleError(e, `Failed to import ${item.name}`);
          }
        }),
      );
      openTabs(resourceTabs(ids), { placement });
    },
    [handleError, openTabs],
  );
};
