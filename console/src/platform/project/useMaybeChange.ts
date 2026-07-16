// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";

export const useMaybeChange = (): ((key: string) => Promise<void>) => {
  const dispatch = useDispatch();
  // Optional: the active project vanishes transiently when it is deleted, and this
  // hook's consumers stay subscribed until the Guard unmounts them.
  const selected = Session.Project.useSelectOptionalSelected();
  const client = Synnax.use();
  return useCallback(
    async (key) => {
      if (selected === key) return;
      if (client == null) throw new DisconnectedError();
      const { layout } = await client.projects.retrieve(key);
      dispatch(Session.Project.select(key));
      dispatch(
        Session.Layout.setProject({ slice: Session.Layout.migrateLayout(layout) }),
      );
    },
    [dispatch, selected, client],
  );
};
