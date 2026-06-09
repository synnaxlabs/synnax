// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { useSelectActiveKey } from "@/workspace/selectors";
import { useMaybeChange } from "@/workspace/useMaybeChange";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

interface CreatedRecord {
  workspace?: workspace.Key;
  key: string;
  name: string;
}

interface UseFluxCreate<Input, Output extends CreatedRecord> {
  (args: {
    afterSuccess: (params: Flux.AfterSuccessParams<Output>) => Promise<void>;
  }): { update: (data: Input) => void };
}

export interface CreateUseCreateArgs<Input, Output extends CreatedRecord> {
  // useCreate is the Pluto flux hook that persists the record on the server.
  useCreate: UseFluxCreate<Input, Output>;
  // createLayout builds the layout placed once the record exists on the server.
  createLayout: (record: Pick<Output, "key" | "name">) => Layout.Creator;
  // defaultName is applied when the caller does not supply a name.
  defaultName: string;
  // useDefaults supplies per-render default fields merged beneath caller overrides.
  useDefaults?: () => Partial<Input>;
}

// createUseCreate builds a useCreate hook for a workspace-scoped layout resource. The
// returned hook creates the record on the server through its Pluto flux store, switches
// to the owning workspace, and places the resource's layout once the record exists, so
// the connected component can retrieve it without a pendingUpload round-trip.
export const createUseCreate =
  <
    Input extends { workspace?: workspace.Key; name: string },
    Output extends CreatedRecord,
  >({
    useCreate,
    createLayout,
    defaultName,
    useDefaults,
  }: CreateUseCreateArgs<Input, Output>) =>
  ({ workspace }: UseCreateProps): ((init?: Partial<Input>) => void) => {
    const activeWorkspace = useSelectActiveKey();
    const maybeChangeWorkspace = useMaybeChange();
    const placeLayout = Layout.usePlacer();
    const defaults = useDefaults?.();
    const { update } = useCreate({
      afterSuccess: async ({ data: { workspace, key, name } }) => {
        if (workspace != null) await maybeChangeWorkspace(workspace);
        placeLayout(createLayout({ key, name }));
      },
    });
    return useCallback(
      (init) =>
        update({
          name: defaultName,
          ...defaults,
          ...init,
          workspace: workspace ?? activeWorkspace ?? undefined,
        } as Input),
      [update, defaults, workspace, activeWorkspace, defaultName],
    );
  };
