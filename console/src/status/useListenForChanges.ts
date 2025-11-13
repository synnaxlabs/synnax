// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { filterFavoritesToKeys, removeFavorites } from "@/status/slice";

export const useListenForChanges = () => {
  const dispatch = useDispatch();
  const addStatus = Status.useAdder();
  const listQuery = Status.useList();
  useAsyncEffect(
    async (signal) => {
      await listQuery.retrieveAsync({}, { signal });
      if (listQuery.variant !== "success") return;
      dispatch(filterFavoritesToKeys(listQuery.data));
    },
    [dispatch],
  );
  Status.useSetSynchronizer(addStatus);
  const handleDelete = useCallback(
    (key: status.Key) => {
      dispatch(removeFavorites(key));
    },
    [dispatch],
  );
  Status.useDeleteSynchronizer(handleDelete);
};
