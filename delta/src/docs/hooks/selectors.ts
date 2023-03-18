// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DocsLocation, DocsState, DocsStoreState, DOCS_SLICE_NAME } from "@/docs/store";
import { useMemoSelect } from "@/hooks";

/** Selects the documentation slice from a larger store state. */
export const selectDocsState = (state: DocsStoreState): DocsState =>
  state[DOCS_SLICE_NAME];

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const selectDocsLocation = (state: DocsStoreState): DocsLocation =>
  selectDocsState(state).location;

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const useSelectDocsLocation = (): DocsLocation =>
  useMemoSelect(selectDocsLocation, []);
