// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DocsLocation, DocsState, DocsStoreState, DOCS_SLICE_NAME } from "./slice";

import { useMemoSelect } from "@/hooks";

export const selectDocsState = (state: DocsStoreState): DocsState =>
  state[DOCS_SLICE_NAME];

export const selectDocsLocation = (state: DocsStoreState): DocsLocation =>
  selectDocsState(state).location;

export const useSelectDocsLocation = (): DocsLocation =>
  useMemoSelect(selectDocsLocation, []);
