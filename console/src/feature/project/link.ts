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
import { Session } from "@/session";

export const useLink = (): Link.Handler => {
  const dispatch = Session.useDispatch();
  return useCallback(
    async ({ client, key }) => {
      const { layout, ...proj } = await client.projects.retrieve(key);
      dispatch(
        Session.Layout.setProject({ slice: layout as Session.Layout.SliceState }),
      );
      dispatch(Session.Project.select(proj.key));
    },
    [dispatch],
  );
};
