// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";

import { deleteToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Logs out of the selected Core: ends its session, forgets the saved token and the
 * password, and clears the session state. The local state clears at once, and the Core
 * side follows; a session the Core cannot be asked to end idles out.
 */
export const useLogout = (): (() => void) => {
  const client = Synnax.use();
  const core = Session.Core.useSelectSelected();
  const dispatch = Session.useDispatch();
  const clearSession = Session.useLogout();
  const handleError = Status.useErrorHandler();
  return () => {
    // the delete goes out before a later login can save a token, so it never removes
    // a newer one
    if (core != null)
      handleError(async () => await deleteToken(core), "Failed to forget the session");
    handleError(async () => await client?.auth.logout(), "Failed to log out");
    clearSession();
    if (core != null) dispatch(Session.Core.set({ ...core, password: "" }));
  };
};
