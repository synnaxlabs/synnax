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
import { useDispatch } from "react-redux";

import { useSelectActiveKey } from "@/project/selectors";
import { setActive } from "@/project/slice";

export const useMaybeChange = (): ((key: string) => Promise<void>) => {
  const dispatch = useDispatch();
  const activeWS = useSelectActiveKey();
  const client = Synnax.use();
  return async (key) => {
    if (activeWS === key) return;
    if (client == null) throw new DisconnectedError();
    const { layout: _layout, ...ws } = await client.projects.retrieve(key);
    dispatch(setActive(ws));
    // Layout loading is no longer a project-switch side effect: panels own the
    // project's tiling via the panel Flux store, and switching projects
    // re-subscribes the panel tab strip automatically.
  };
};
