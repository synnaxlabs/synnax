// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";

import { moveToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Sets the username the selected Core logs in with, after the logged-in user renames
 * their own username. On desktop the saved session moves with it, so the user stays
 * logged in. A Core that logs in with OpenID Connect keeps its record. When the
 * keychain fails, the record stays as it was and the error goes to the status handler.
 */
export const useSetUsername = (): ((username: string) => void) => {
  const core = Session.Core.useSelectSelected();
  const dispatch = Session.useDispatch();
  const handleError = Status.useErrorHandler();
  return (username) => {
    if (core == null || core.username === "") return;
    const renamed = { ...core, username };
    handleError(async () => {
      // the token moves first, so the renamed record finds it
      await moveToken(core, renamed);
      dispatch(Session.Core.set(renamed));
    }, "Failed to move the saved session");
  };
};
