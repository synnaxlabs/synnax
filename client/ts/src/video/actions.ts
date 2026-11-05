// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type Handlers,
  rename,
  setChannel,
  setRange,
} from "@/video/actions.gen";

const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.video);
    return { inverse: [], targets: [payload.video.key] };
  },

  rename: (state, payload) => {
    const prev = state.name;
    state.name = payload.name;
    return { inverse: [rename({ name: prev })], targets: [state.key] };
  },

  setChannel: (state, payload) => {
    const prev = state.channel;
    if (prev === payload.channel) return actions.NO_OP_RESULT;
    state.channel = payload.channel;
    return { inverse: [setChannel({ channel: prev })], targets: [state.key] };
  },

  setRange: (state, payload) => {
    const prev = actions.snapshotDraft(state.range);
    state.range = payload.range;
    return { inverse: [setRange({ range: prev })], targets: [state.key] };
  },
};

export const reduceAll = createReduceAll(handlers);

// createOf hands the dispatch controller the document of a create action, so a
// broadcast for a view that is not cached yet adds it instead of dropping it.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.video : undefined;

// kindOf keys each action by its type, so rapid edits of one kind coalesce into one
// undoable. A batch of actions is one "transaction" undoable.
export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.length > 1) return "transaction";
  return actions[0].type;
};
