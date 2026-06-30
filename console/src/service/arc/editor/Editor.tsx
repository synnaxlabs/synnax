// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { Graph } from "@/service/arc/editor/Graph";
import { Text } from "@/service/arc/editor/Text";
import { Layout } from "@/layout";

const Internal: Layout.Renderer = (props) => {
  const mode = Arc.useSelectMode();
  const C = mode === "graph" ? Graph : Text;
  return <C {...props} />;
};

export const Editor: Layout.Renderer = (props) => (
  <Arc.Suspended arcKey={props.layoutKey}>
    <Internal {...props} />
  </Arc.Suspended>
);

Editor.useName = Layout.createUseFluxName(Arc.useRename, Arc.useRetrieveObservableName);
