// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, panel } from "@synnaxlabs/client";
import { Access, Arc, Icon, Panel as Base, Status } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useCreateModal } from "@/arc/editor/CreateModal";
import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectMode, useSelectVersion } from "@/arc/selectors";
import { internalCreate, type State, ZERO_STATE } from "@/arc/slice";
import { TYPE } from "@/arc/types";
import { translateGraphToConsole, translateGraphToServer } from "@/arc/types/translate";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import { type Tabs } from "@/panel/tabs/index";
import { Selector } from "@/selector";

export const useLoadRemote = createLoadRemote<arc.Arc>({
  useRetrieve: Arc.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) =>
    internalCreate({
      version: ZERO_STATE.version,
      key: v.key,
      remoteCreated: true,
      graph: translateGraphToConsole(v.graph),
      text: v.text,
      mode: v.mode,
    }),
});

const Loaded: Tabs.Content = () => {
  const { layoutKey } = props;
  const mode = useSelectMode(layoutKey) ?? "graph";
  if (mode === "graph") return <Graph.Editor {...props} />;
  return <Text.Editor {...props} />;
};

export const Editor = () => {
  const arc = useLoadRemote();
  if (arc == null) return null;
  return <Loaded {...props} />;
};

Editor.useName = Layout.createUseFluxName(Arc.useRename, Arc.useRetrieveObservableName);
Editor.icon = <Icon.Arc />;

export type CreateArg = Partial<State> & Partial<Tabs.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const {
      name = "Arc Editor",
      location = "mosaic",
      tab,
      mode = "graph",
      ...rest
    } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, mode }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const Selectable: Selector.Selectable = ({ tabKey }) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArcModal = useCreateModal();
  const dispatch = useDispatch();
  const place = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  const panelKey = Layout.useSelectActivePanelKey();
  const { dispatch: panelDispatch } = Base.useDispatch();
  const { update } = Arc.useCreate({
    afterSuccess: useCallback(
      async ({ data: { key } }) => {
        if (tabKey == null || panelKey == null) return;
        panelDispatch({
          key: panelKey,
          actions: [
            panel.setTabType({ key: tabKey, type: arc.ontologyID(key).type }),
            panel.setTabArgs({ key: tabKey, args: { resourceKey: key } }),
          ],
        });
      },
      [tabKey, panelKey, panelDispatch],
    ),
  });

  const handleClick = useCallback(() => {
    handleError(async () => {
      const result = await createArcModal({});
      if (result == null) return;
      const key = uuid.create();
      // In a panel, create the arc on the server (so the tab references a real
      // resource) and seed the local editor's working copy, then fill the tab;
      // otherwise open it as a mosaic tab as before.
      if (tabKey != null && panelKey != null) {
        const zero = deep.copy(ZERO_STATE);
        dispatch(internalCreate({ ...zero, key, mode: result.mode }));
        update({
          key,
          name: result.name,
          mode: result.mode,
          graph: translateGraphToServer(zero.graph),
          text: zero.text,
        });
      } else place(create({ key, name: result.name, mode: result.mode }));
    }, "Failed to create Arc program");
  }, [tabKey, panelKey, place, dispatch, createArcModal, handleError, update]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
