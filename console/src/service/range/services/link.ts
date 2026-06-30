// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { type Link } from "@/service/link";
import { Layout } from "@/layout";
import { Range } from "@/range";

export const useLink = (): Link.Handler => {
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  return useCallback(
    async ({ client, key }) => {
      const range = await client.ranges.retrieve(key);
      dispatch(Range.add({ ranges: Range.fromClientRange(range) }));
      dispatch(Range.select(range.key));
      placeLayout({ ...Range.OVERVIEW_LAYOUT, key, name: range.name });
    },
    [dispatch, placeLayout],
  );
};
