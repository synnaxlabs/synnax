// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query } from "@synnaxlabs/client";

import { type Project } from "@/session/project";
import { Synchronizer } from "@/session/synchronizer";
import {
  type Action,
  removeTabs,
  removeViews,
  type StoreState,
} from "@/session/view/slice";

interface RequiredStoreState extends StoreState, Project.StoreState {}

// A closed tab keeps its entry until the next sweep, as nothing reads it before then.
const removeClosedTabs: Synchronizer.Callbacks<RequiredStoreState, Action> = {
  reconcile: async ({ client, store, ended }) => {
    const { view, project } = store.getState();
    const tabs = Object.keys(view.open);
    const projectKey = project.selected;
    const user = client.users.current.watch().getCached();
    if (tabs.length === 0 || projectKey == null || !query.isLive(user)) return;
    const panels = await client.panels.retrieve({
      where: panel.visibleTo(projectKey, user.key),
    });
    if (ended() || store.getState().project.selected !== projectKey) return;
    const live = new Set(
      panels.flatMap(({ root }) => panel.leafTabGroups(root).flat()),
    );
    const closed = tabs.filter((tab) => !live.has(tab));
    if (closed.length > 0) store.dispatch(removeTabs({ keys: closed }));
  },
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<RequiredStoreState, Action> = [
  Synchronizer.createRemover<StoreState, Action>({
    name: "remove deleted views from explorer tabs",
    domain: (client) => client.views,
    selectKeys: (state) => [...new Set(Object.values(state.view.open))],
    remove: (keys) => removeViews({ keys }),
  }),
  { name: "remove closed explorer tabs", use: () => removeClosedTabs },
];
