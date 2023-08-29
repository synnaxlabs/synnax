// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";

import { DocsLocation, DocsState, DocsStoreState, DOCS_SLICE_NAME } from "@/docs/store";

/** Selects the documentation slice from a larger store state. */
export const selectSliceState = (state: DocsStoreState): DocsState =>
  state[DOCS_SLICE_NAME];

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const selectLocation = (state: DocsStoreState): DocsLocation =>
  selectSliceState(state).location;

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const useSelectLocation = (): DocsLocation => useMemoSelect(selectLocation, []);
