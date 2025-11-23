// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type Synnax as Client } from "@synnaxlabs/client";
import { Flux, type Pluto, Status, Synnax, Workspace } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { purgeExcludedLayouts } from "@/workspace/purgeExcludedLayouts";
import { useSelectActive } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

export const useCreateOrRetrieve = () => {
  const handleError = Status.useErrorHandler();
  const dispatch = useDispatch();
  const prevClient = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const layout = Layout.useSelectSliceState();
  const activeWS = useSelectActive();
  return (client: Client) => {
    if (activeWS == null) return;
    const purgedLayout = purgeExcludedLayouts(layout);
    if (prevClient != null && Workspace.editAccessGranted({ key: activeWS.key, store: fluxStore, client: prevClient }))
      handleError(
        async () => await prevClient.workspaces.setLayout(activeWS.key, purgedLayout),
        `Failed to save workspace ${activeWS.name} to ${prevClient.params.name ?? "previous Core"}`,
      );
    handleError(
      async () => {
        try {
          await client.workspaces.retrieve(activeWS.key);
          if (Workspace.editAccessGranted({ key: activeWS.key, store: fluxStore, client }))
            await client.workspaces.setLayout(activeWS.key, purgedLayout);
          dispatch(setActive(activeWS));
        } catch (e) {
          if (!NotFoundError.matches(e)) throw e;
          await client.workspaces.create({ ...activeWS, layout: purgedLayout });
          dispatch(setActive(activeWS));
        }
      },
      `Failed to create workspace ${activeWS.name} on ${client.params.name ?? "current Core"}`,
    );
  };
};
