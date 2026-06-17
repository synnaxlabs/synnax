// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul } from "@synnaxlabs/pluto";

import { SLICE_NAME, type StoreState } from "@/hauling/slice";
import { useMemoSelect } from "@/hooks";

export const selectHauling = (state: StoreState): Haul.DraggingState =>
  state[SLICE_NAME];

export const useSelectHauling = (): Haul.DraggingState =>
  useMemoSelect(selectHauling, []);
