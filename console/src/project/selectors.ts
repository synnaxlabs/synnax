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

export const selectActive = (state: StoreState): Project | null =>
  selectState(state).active;

export const useSelectActive = (): Project | null => useMemoSelect(selectActive, []);

export const selectActiveKey = (state: StoreState): project.Key | null =>
  selectState(state).active?.key ?? null;

export const useSelectActiveKey = (): project.Key | null =>
  useMemoSelect(selectActiveKey, []);

export const selectActiveName = (state: StoreState): string | null =>
  selectState(state).active?.name ?? null;

export const useSelectActiveName = (): string | null =>
  useMemoSelect(selectActiveName, []);
