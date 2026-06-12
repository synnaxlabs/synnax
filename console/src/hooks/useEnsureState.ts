// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

export interface CreateEnsureStateParams {
  useExists: (key: string) => boolean;
  create: (key: string) => PayloadAction<{ key: string }>;
}

/**
 * Creates a hook that guarantees a per-layout entry exists in a visualization's Redux
 * slice before its renderer reads from it. The layout creator initializes this entry
 * when a visualization is placed directly, but layouts restored from a workspace bypass
 * the creator, so the renderer must initialize it on mount. The hook returns whether the
 * entry
 * currently exists; callers should withhold rendering the visualization until it does.
 */
export const createEnsureState =
  ({ useExists, create }: CreateEnsureStateParams) =>
  (key: string): boolean => {
    const dispatch = useDispatch();
    const exists = useExists(key);
    useEffect(() => {
      if (!exists) dispatch(create(key));
    }, [dispatch, exists, key, create]);
    return exists;
  };
