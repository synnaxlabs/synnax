// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { useDataUtils } from "@/list/Data";

export interface UseSynchronizerProps<K extends record.Key, E extends record.Keyed<K>> {
  useSetSynchronizer?: (onSet: (e: E) => void) => void;
  useDeleteSynchronizer?: (onDelete: (k: K) => void) => void;
}

export const useSynchronizer = <K extends record.Key, E extends record.Keyed<K>>({
  useSetSynchronizer,
  useDeleteSynchronizer,
}: UseSynchronizerProps<K, E>): void => {
  const { setSourceData } = useDataUtils<K, E>();
  useSetSynchronizer?.((e) => setSourceData((prev) => [...prev, e]));
  useDeleteSynchronizer?.((k) =>
    setSourceData((prev) => prev.filter((p) => p.key !== k)),
  );
};

export const Synchronizer = <K extends record.Key, E extends record.Keyed<K>>({
  useSetSynchronizer,
  useDeleteSynchronizer,
}: UseSynchronizerProps<K, E>): null => {
  useSynchronizer({ useSetSynchronizer, useDeleteSynchronizer });
  return null;
};
