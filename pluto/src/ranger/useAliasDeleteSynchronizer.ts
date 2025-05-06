// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MultiSeries, ranger } from "@synnaxlabs/client";
import { useCallback } from "react";

import { Synch } from "@/synch";

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void => {
  const handleUpdate = useCallback(
    ({ series }: MultiSeries) =>
      series
        .flatMap((s) => s.toStrings())
        .map(ranger.decodeDeleteAliasChange)
        .forEach(onDelete),
    [onDelete],
  );
  Synch.useListener(ranger.DELETE_ALIAS_CHANNEL_NAME, handleUpdate);
};
