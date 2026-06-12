// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type workspace } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { useSelectActiveKey } from "@/workspace/selectors";
import { useMaybeChange } from "@/workspace/useMaybeChange";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

interface CreatedRecord {
  key: string;
  name: string;
}

interface UseFluxCreate<Input, Output extends CreatedRecord> {
  (args: {
    afterSuccess: (params: Flux.AfterSuccessParams<Output>) => Promise<void>;
  }): { update: (data: Input) => void };
}

// Everything toCreateParams needs to assemble the create body: the caller's overrides,
// the resolved workspace, and the store for deriving default fields.
export interface ToCreateParams<Input> {
  // overrides are the caller-supplied fields passed to the returned hook.
  overrides?: Partial<Input>;
  // workspace is the resolved target workspace, if any.
  workspace?: workspace.Key;
  // store exposes redux state for callers that derive default fields from it.
  store: Store<RootState>;
}

export interface CreateUseCreateArgs<Input, Output extends CreatedRecord> {
  // useCreate is the Pluto flux hook that persists the record on the server.
  useCreate: UseFluxCreate<Input, Output>;
  // createLayout builds the layout placed once the record exists on the server.
  createSessionState: (record: Pick<Output, "key" | "name">) => Layout.Creator;
  // toCreateParams assembles the create body, including the resource's default name and
  // any per-render default fields. Constructed at the concrete call site so Input needs
  // no cast; a caller override in overrides wins by spreading over the defaults.
  toCreateParams: (params: ToCreateParams<Input>) => Input;
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
    createSessionState,
    toCreateParams,
  }: CreateUseCreateArgs<Input, Output>) =>
  ({ workspace }: UseCreateProps): ((init?: Partial<Input>) => void) => {
    const activeWorkspace = useSelectActiveKey();
    const maybeChangeWorkspace = useMaybeChange();
    const placeLayout = Layout.usePlacer();
    const store = useStore<RootState>();
    workspace ??= activeWorkspace ?? undefined;
    const { update } = useCreate({
      afterSuccess: useCallback(
        async ({ data: { key, name } }) => {
          if (workspace != null) await maybeChangeWorkspace(workspace);
          placeLayout(createSessionState({ key, name }));
        },
        [workspace],
      ),
    });
    return useCallback(
      (overrides) => update(toCreateParams({ overrides, workspace, store })),
      [update, store, workspace, toCreateParams],
    );
  };
