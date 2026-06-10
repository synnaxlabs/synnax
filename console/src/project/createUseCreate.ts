// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type ontology, type project } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { useSelectActiveKey } from "@/project/selectors";
import { useMaybeChange } from "@/project/useMaybeChange";
import { type RootState } from "@/store";

export interface UseCreateProps {
  project?: project.Key;
  // onResolved, when provided, replaces opening the resource as a mosaic tab: the
  // created record is handed back as a resource to fill a panel tab in place.
  onResolved?: (content: { resource: ontology.ID }) => void;
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
// the resolved project, and the store for deriving default fields.
export interface ToCreateParams<Input> {
  // overrides are the caller-supplied fields passed to the returned hook.
  overrides?: Partial<Input>;
  // project is the resolved target project, if any.
  project?: project.Key;
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
  // ontologyID maps the created record's key to the resource handed to onResolved.
  ontologyID: (key: string) => ontology.ID;
}

// createUseCreate builds a useCreate hook for a project-scoped layout resource. The
// returned hook creates the record on the server through its Pluto flux store, then
// either resolves it into a panel tab (when onResolved is provided) or switches to the
// owning project and places the resource's layout, so the connected component can
// retrieve it without a pendingUpload round-trip.
export const createUseCreate =
  <
    Input extends { project?: project.Key; name: string },
    Output extends CreatedRecord,
  >({
    useCreate,
    createSessionState,
    toCreateParams,
    ontologyID,
  }: CreateUseCreateArgs<Input, Output>) =>
  ({ project, onResolved }: UseCreateProps): ((init?: Partial<Input>) => void) => {
    const activeProject = useSelectActiveKey();
    const maybeChangeProject = useMaybeChange();
    const placeLayout = Layout.usePlacer();
    const store = useStore<RootState>();
    project ??= activeProject ?? undefined;
    const { update } = useCreate({
      afterSuccess: useCallback(
        async ({ data: { key, name } }) => {
          if (onResolved != null) {
            onResolved({ resource: ontologyID(key) });
            return;
          }
          if (project != null) await maybeChangeProject(project);
          placeLayout(createSessionState({ key, name }));
        },
        [project, onResolved],
      ),
    });
    return useCallback(
      (overrides) => update(toCreateParams({ overrides, project, store })),
      [update, store, project, toCreateParams],
    );
  };
