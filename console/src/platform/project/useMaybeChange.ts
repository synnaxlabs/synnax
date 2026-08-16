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

import { Session } from "@/session";

/**
 * useMaybeChange returns a callback that switches the active project to the given
 * key when it is not already active. Panels are per-project documents fetched from
 * the core, so switching only updates the selection; the panel selector reconciles
 * the window's selected panel against the new project's panels.
 */
export const useMaybeChange = (): ((key: string) => void) => {
  const dispatch = useDispatch();
  // Optional: the active project vanishes transiently when it is deleted, and this
  // hook's consumers stay subscribed until the Guard unmounts them.
  const selected = Session.Project.useSelectOptionalSelected();
  return useCallback(
    (key) => {
      if (selected === key) return;
      dispatch(Session.Project.select(key));
    },
    [dispatch, selected],
  );
};
