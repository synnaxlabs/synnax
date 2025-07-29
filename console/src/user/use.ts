// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user as clientUser } from "@synnaxlabs/client";
import { Unreachable } from "@synnaxlabs/freighter";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { giveAll, set } from "@/permissions/slice";
import { clear as clearUser, set as setUser } from "@/user/slice";

export const useRetrieve = (): void => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        dispatch(giveAll());
        dispatch(clearUser());
        return;
      }
      const username = client.props.username;
      try {
        const user = await client.user.retrieveByName(username);
        if (signal.aborted) return;

        // Store the user data
        dispatch(setUser({ user }));

        // Retrieve and store permissions
        const policies = await client.access.policy.retrieveFor(
          clientUser.ontologyID(user.key),
        );
        if (signal.aborted) return;
        dispatch(set({ policies }));
      } catch (e) {
        if (Unreachable.matches(e)) return;
        handleError(e, `Failed to fetch permissions for ${username}`);
        // Clear user data on error
        dispatch(clearUser());
      }
    },
    [client],
  );
};
