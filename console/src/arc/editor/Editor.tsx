// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc, Icon } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useCreateModal } from "@/arc/editor/CreateModal";
import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectMode, useSelectVersion } from "@/arc/selectors";
import { internalCreate, type State, ZERO_STATE } from "@/arc/slice";
import { TYPE } from "@/arc/types";
import { translateGraphToConsole } from "@/arc/types/translate";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
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

const Loaded: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  const mode = useSelectMode(layoutKey) ?? "graph";
  if (mode === "graph") return <Graph.Editor {...props} />;
  return <Text.Editor {...props} />;
};

export const Editor: Layout.Renderer = (props) => {
  const arc = useLoadRemote(props.layoutKey);
  if (arc == null) return null;
  return <Loaded {...props} />;
};

Editor.useName = Layout.createUseFluxName(Arc.useRename, Arc.useRetrieveObservableName);
Editor.icon = <Icon.Arc />;

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

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

export const Selectable: Selector.Selectable = ({
  layoutKey,
  onPlace,
  onResolved,
  handleError,
}) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArcModal = useCreateModal();
  const dispatch = useDispatch();

  const handleClick = useCallback(() => {
    handleError(async () => {
      const result = await createArcModal({});
      if (result == null) return;
      // In a panel, create the local arc state (so it renders and later syncs to the
      // server on edit) and fill the tab with its resource; otherwise open it as a
      // mosaic tab as before.
      if (onResolved != null) {
        dispatch(
          internalCreate({
            ...deep.copy(ZERO_STATE),
            key: layoutKey,
            mode: result.mode,
          }),
        );
        onResolved({ resource: arc.ontologyID(layoutKey) });
      } else
        onPlace(create({ key: layoutKey, name: result.name, mode: result.mode }));
    }, "Failed to create Arc program");
  }, [onResolved, onPlace, layoutKey, dispatch, createArcModal, handleError]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
