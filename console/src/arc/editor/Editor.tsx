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

import { useCreateModal } from "@/arc/editor/CreateModal";
import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectMode, useSelectVersion } from "@/arc/selectors";
import { internalCreate, type State, ZERO_STATE } from "@/arc/slice";
import { translateGraphToConsole } from "@/arc/types/translate";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { type Layout } from "@/layout";
import { Selector } from "@/selector";

export const useLoadRemote = createLoadRemote<arc.Arc>({
  useRetrieve: Arc.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) =>
    internalCreate({
      version: "0.0.0",
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
export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const LAYOUT_TYPE = "arc_editor";
export type LayoutType = typeof LAYOUT_TYPE;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Arc Editor", location = "mosaic", tab, mode, ...rest } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, mode }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const Selectable: Selector.Selectable = ({
  layoutKey,
  onPlace,
  handleError,
}) => {
  const visible = Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArcModal = useCreateModal();

  const handleClick = useCallback(() => {
    handleError(async () => {
      const result = await createArcModal({});
      if (result != null)
        onPlace(create({ key: layoutKey, name: result.name, mode: result.mode }));
    }, "Failed to create Arc program");
  }, [onPlace, layoutKey, createArcModal, handleError]);

  if (!visible) return null;

  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);
