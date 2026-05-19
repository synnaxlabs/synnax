// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Status as PStatus } from "@synnaxlabs/pluto/status";
import { Synnax } from "@synnaxlabs/pluto/synnax";
import { useCallback, useEffectEvent } from "react";
import { useDispatch } from "react-redux";

import { filterFavoritesToKeys, removeFavorites } from "@/status/slice";

export const useListenForChanges = () => {
  const dispatch = useDispatch();
  const addStatus = Status.useAdder();
  const listQuery = PStatus.useList();
  const client = Synnax.use();
  const onVariantChange = useEffectEvent(() => {
    if (listQuery.variant !== "success") return;
    dispatch(filterFavoritesToKeys(listQuery.data));
  });
  useAsyncEffect(
    async (signal) => {
      await listQuery.retrieveAsync({}, { signal });
      onVariantChange();
    },
    [listQuery.retrieveAsync, client?.key],
  );
  PStatus.useSetSynchronizer(addStatus);
  const handleDelete = useCallback(
    (key: status.Key) => {
      dispatch(removeFavorites(key));
    },
    [dispatch],
  );
  PStatus.useDeleteSynchronizer(handleDelete);
};
