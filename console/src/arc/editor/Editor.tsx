// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Graph } from "@/arc/editor/graph";
import { useLoadRemote } from "@/arc/editor/hooks";
import { ModeSelector } from "@/arc/editor/ModeSelector";
import { Text } from "@/arc/editor/text";
import { useSelectMode } from "@/arc/selectors";
import { setMode } from "@/arc/slice";
import { type Layout } from "@/layout";

const Loaded: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  const mode = useSelectMode(layoutKey);
  const dispatch = useDispatch();

  const handleModeSelect = useCallback(
    (selectedMode: arc.Mode) =>
      dispatch(setMode({ key: layoutKey, mode: selectedMode })),
    [dispatch, layoutKey],
  );

  if (mode == null) return <ModeSelector onSelect={handleModeSelect} />;

  if (mode === "graph") return <Graph.Editor {...props} />;
  return <Text.Editor {...props} />;
};

export const Editor: Layout.Renderer = (props) => {
  const arc = useLoadRemote(props.layoutKey);
  if (arc == null) return null;
  return <Loaded {...props} />;
};
