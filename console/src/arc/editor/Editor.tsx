// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectExists } from "@/arc/selectors";
import { internalCreate } from "@/arc/slice";
import { createEnsureState } from "@/hooks/useEnsureState";
import { Layout } from "@/layout";

const Loaded: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  Arc.useEnsureRetrieved({ key: layoutKey });
  const mode = Arc.useSelectMode({ key: layoutKey });
  if (mode === "graph") return <Graph.Editor {...props} />;
  return <Text.Editor {...props} />;
};

const useEnsureState = createEnsureState({
  useExists: useSelectExists,
  create: (key) => internalCreate({ key }),
});

export const Editor: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  const exists = useEnsureState(layoutKey);
  if (!exists) return null;
  return <Loaded {...props} />;
};

Editor.useName = Layout.createUseFluxName(Arc.useRename, Arc.useRetrieveObservableName);
