// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, workspace } from "@synnaxlabs/client";
import { Flux, Ontology, type Pluto, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";

import { useSelectActiveKey } from "@/workspace/selectors";

// Adopts a resource that was created without a workspace into the active workspace.
//
// Pluto-synced visualizations (e.g. tables) are written to the server as soon as they
// are created, even with no active workspace, leaving them without a workspace ParentOf
// relationship. Nothing re-sends them once a workspace is later created, so they would
// otherwise never become part of it. Mounting this hook in the visualization's renderer
// links the orphaned resource to the active workspace the moment one exists.
export const useAdoptIntoActiveWorkspace = (id: ontology.ID): void => {
  const client = Synnax.use();
  const store = Flux.useStore<Pluto.FluxStore>();
  const activeWorkspace = useSelectActiveKey();
  useAsyncEffect(
    async (signal) => {
      if (client == null || activeWorkspace == null) return;
      // A cached parent is authoritative: the store only holds relationships that
      // exist on the server. A cache miss is ambiguous, though - a true orphan and a
      // cold store after reconnect look identical - so confirm against the server
      // before adopting, otherwise a reconnect could attach a second workspace parent.
      if (Ontology.retrieveCachedParentID(store, id) != null) return;
      const parents = await client.ontology.retrieveParents(id);
      if (signal.aborted || parents.length > 0) return;
      await client.ontology.addChildren(workspace.ontologyID(activeWorkspace), id);
    },
    [client, store, activeWorkspace, ontology.idToString(id)],
  );
};
