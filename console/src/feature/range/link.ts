// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { type Link } from "@/feature/link";
import { Range } from "@/platform/range";
import { Session } from "@/session";

export const useLink = (): Link.Handler => {
  const dispatch = Session.useDispatch();
  const placeLayout = Session.Layout.usePlacer();
  return useCallback(
    async ({ client, key }) => {
      const range = await client.ranges.retrieve(key);
      Session.Range.fromClient(range).forEach((r) => dispatch(Session.Range.add(r)));
      dispatch(Session.Range.select(range.key));
      placeLayout({ ...Range.OVERVIEW_LAYOUT, key, name: range.name });
    },
    [dispatch, placeLayout],
  );
};
