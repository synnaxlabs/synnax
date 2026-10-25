// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type panel } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";

import { Panel } from "@/platform/panel";

/** Maps the resources importItem created to the tabs that open them. */
const resourceTabs = (ids: (ontology.ID | void)[]): panel.NewTab[] =>
  ids.filter((id) => id != null).map((resource) => ({ variant: "resource", resource }));

export interface ImportBatchParams<I extends { name: string }> {
  /** The items to import. An item's name identifies it when its import fails. */
  items: I[];
  /** Creates the resource the item describes, or nothing when it opens no tab. TODO:
   * should probably NOT return `void`. */
  importItem: (item: I) => Promise<void | ontology.ID>;
  /** Places the tabs in a specific leaf instead of beside the current one. */
  placement?: Panel.Placement;
  /** Called for each item whose import succeeds. */
  onSuccess?: (item: I) => void;
}

export interface ImportBatch {
  <I extends { name: string }>(params: ImportBatchParams<I>): Promise<void>;
}

const importOne = async <I extends { name: string }>(
  item: I,
  importItem: ImportBatchParams<I>["importItem"],
  handleError: Status.ErrorHandler,
  onSuccess?: ImportBatchParams<I>["onSuccess"],
): Promise<void | ontology.ID> => {
  try {
    const id = await importItem(item);
    onSuccess?.(item);
    return id;
  } catch (e) {
    handleError(e, { message: "Failed to import", name: item.name });
  }
};

/**
 * Returns a batch importer: it imports every item concurrently, then opens the
 * resources they created as one batch of tabs. An item that fails is reported on its
 * own and leaves the rest of the batch running.
 */
export const useImportBatch = (): ImportBatch => {
  const handleError = Status.useErrorHandler();
  const openTabs = Panel.useOpenTabs();
  return async <I extends { name: string }>({
    items,
    importItem,
    placement,
    onSuccess,
  }: ImportBatchParams<I>): Promise<void> => {
    const ids = await Promise.all(
      items.map((item) => importOne(item, importItem, handleError, onSuccess)),
    );
    openTabs(resourceTabs(ids), { placement });
  };
};
