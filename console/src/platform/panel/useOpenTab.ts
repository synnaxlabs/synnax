// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type Flux, Panel } from "@synnaxlabs/pluto";
import { type optional, uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Session } from "@/session";

export type OpenTabParams =
  | optional.Optional<panel.TabResource, "key">
  | optional.Optional<panel.TabView, "key">;

export type OpenTab = (params: OpenTabParams) => void;

export const useOpenTab = (): ((params: OpenTabParams) => void) => {
  const dispatchSession = Session.useDispatch();
  const { dispatch } = Panel.useDispatch();
  const selectTab = Session.Panel.useSelectTab();
  const parentPanelKey = Panel.useOptionalKey();
  const getSelected = Session.Panel.useGetSelected();
  const parentTabKey = Panel.useOptionalTabKey();
  const handleTabCreate = useCallback(
    (params: OpenTabParams) => {
      const panelKey = parentPanelKey ?? getSelected();
      if (panelKey == null) return;
      const tab: panel.Tab = { key: parentTabKey ?? uuid.create(), ...params };
      dispatch({ key: panelKey, actions: [panel.insertTab({ tab })] });
      selectTab(tab.key);
    },
    [parentTabKey, parentPanelKey, dispatch, dispatchSession],
  );
  const { update: createPanel } = Panel.useCreate<OpenTabParams>({
    beforeUpdate: useCallback(
      ({
        extra,
      }: Flux.BeforeUpdateParams<
        panel.New,
        false,
        Panel.FluxSubStore,
        OpenTabParams
      >) => {
        const anySelected = getSelected() != null;
        if (!anySelected && parentPanelKey == null) return true;
        handleTabCreate(extra);
        return false;
      },
      [parentPanelKey],
    ),
    afterOptimistic: useCallback(
      ({
        data: { key },
        rollbacks,
        extra,
      }: Flux.AfterOptimisticParams<
        panel.Panel,
        false,
        Panel.FluxSubStore,
        OpenTabParams
      >) => {
        dispatchSession(Session.Panel.select({ key }));
        rollbacks.push(() => dispatchSession(Session.Panel.clearSelected({})));
        handleTabCreate(extra);
      },
      [dispatchSession],
    ),
  });
  return useCallback(
    (tab: OpenTabParams) => createPanel({ name: "New Panel" }, { extra: tab }),
    [createPanel],
  );
};
