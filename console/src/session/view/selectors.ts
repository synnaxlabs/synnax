// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, type view } from "@synnaxlabs/client";

import { Select } from "@/session/select";
import { type State } from "@/session/store";
import { SLICE_NAME } from "@/session/view/slice";

const selectOpen = (state: State, tab: panel.TabKey): view.Key | undefined =>
  state[SLICE_NAME].open[tab];

/** Selects the saved view the tab shows, or undefined when it shows its static view. */
export const useSelectOpen = (tab: panel.TabKey): view.Key | undefined =>
  Select.useMemo((state: State) => selectOpen(state, tab), [tab]);
