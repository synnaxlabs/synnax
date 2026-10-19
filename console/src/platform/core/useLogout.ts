// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { useCallback } from "react";

import { deleteToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Logs out of the selected Core. Clearing the selection disposes the client, and
 * closing it ends the session on the Core; the saved token and the password go too.
 */
export const useLogout = (): (() => void) => {
  const core = Session.Core.useSelectSelected();
  const dispatch = Session.useDispatch();
  const clearSession = Session.useLogout();
  const handleError = Status.useErrorHandler();
  return useCallback(() => {
    clearSession();
    if (core == null) return;
    dispatch(Session.Core.set({ ...core, password: "" }));
    handleError(async () => await deleteToken(core), "Failed to forget the session");
  }, [core, dispatch, clearSession, handleError]);
};
