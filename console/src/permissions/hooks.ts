// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Unreachable } from "@synnaxlabs/freighter";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { giveAll, set } from "@/permissions/slice";

export const useFetchPermissions = async (): Promise<void> => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const addStatus = Status.useAggregator();
  useAsyncEffect(async () => {
    if (client == null) {
      dispatch(giveAll());
      return;
    }
    const username = client.props.username;
    try {
      const user_ = await client.user.retrieveByName(username);
      const policies = await client.access.policy.retrieveFor(
        user.ontologyID(user_.key),
      );
      dispatch(set({ policies }));
    } catch (e) {
      if (Unreachable.matches(e)) return;
      Status.handleException(
        e,
        `Failed to fetch permissions for ${username}`,
        addStatus,
      );
    }
  }, [client]);
};
