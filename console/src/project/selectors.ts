// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type StoreState } from "@/project/slice";
import { type Project, type SliceState } from "@/project/types";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

// The default selectors assume an active project, which Project.Guard guarantees for
// everything it wraps. Callers that can observe the no-project state - the guard
// itself, the create flow, the project-delete handler, and deep-link handlers that run
// before a project is chosen - must use the Optional variants.

export const selectActive = (state: StoreState): Project =>
  selectState(state).active as Project;

export const useSelectActive = (): Project => useMemoSelect(selectActive, []);

export const selectActiveKey = (state: StoreState): project.Key =>
  selectActive(state).key;

export const useSelectActiveKey = (): project.Key => useMemoSelect(selectActiveKey, []);

export const selectActiveName = (state: StoreState): string => selectActive(state).name;

export const useSelectActiveName = (): string => useMemoSelect(selectActiveName, []);

export const selectOptionalActive = selectActive as (
  state: StoreState,
) => Project | null;

export const useSelectOptionalActive = (): Project | null =>
  useMemoSelect(selectOptionalActive, []);

export const selectOptionalActiveKey = (state: StoreState): project.Key | null =>
  selectOptionalActive(state)?.key ?? null;

export const useSelectOptionalActiveKey = (): project.Key | null =>
  useMemoSelect(selectOptionalActiveKey, []);

export const selectOptionalActiveName = (state: StoreState): string | null =>
  selectOptionalActive(state)?.name ?? null;

export const useSelectOptionalActiveName = (): string | null =>
  useMemoSelect(selectOptionalActiveName, []);
