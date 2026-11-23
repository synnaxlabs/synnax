// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { rack } from "@synnaxlabs/client";

import { SLICE_NAME, type StoreState } from "@/session/file-browser/slice";
import { Select } from "@/session/select";

const NO_FOLDERS: string[] = [];

const selectRecent = (state: StoreState, rack: rack.Key): string[] =>
  state[SLICE_NAME].recent[rack] ?? NO_FOLDERS;

/** @returns the folders of the files picked on the rack, newest first. */
export const useSelectRecent = (rack: rack.Key): string[] =>
  Select.useMemo((state: StoreState) => selectRecent(state, rack), [rack]);
