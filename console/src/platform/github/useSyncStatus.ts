// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";

import { retrieveStatus } from "@/platform/github/star";
import { Session } from "@/session";

/**
 * Asks the GitHub CLI whether the user starred Synnax, and stores the answer for every
 * star button. Does nothing outside Synnax Desktop or once the user starred Synnax.
 */
export const useSyncStatus = (): void => {
  const dispatch = Session.useDispatch();
  const starred = Session.GitHub.useSelectStarred();
  useAsyncEffect(
    async (signal) => {
      if (!DESKTOP || starred) return;
      const status = await retrieveStatus();
      if (signal.aborted) return;
      if (status === "starred") dispatch(Session.GitHub.markStarred());
      else
        dispatch(
          Session.GitHub.setCLI(status === "not_starred" ? "ready" : "unavailable"),
        );
    },
    [starred],
  );
};
