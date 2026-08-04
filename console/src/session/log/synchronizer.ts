// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, remove, type StoreState } from "@/session/log/slice";
import { Synchronizer } from "@/session/synchronizer";

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = {
  useRemoveDeletedLogs: Synchronizer.createRemover<StoreState, Action>({
    domain: (client) => client.logs,
    selectKeys: (state: StoreState) => Object.keys(state.log.logs),
    remove: (keys) => remove({ keys }),
  }),
};
