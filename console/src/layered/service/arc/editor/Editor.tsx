// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc as Base } from "@synnaxlabs/pluto";

import { Editor as GraphEditor } from "@/layered/service/arc/editor/graph/Editor";
import { Editor as TextEditor } from "@/layered/service/arc/editor/text/Editor";
import { Layout } from "@/layout";

const Internal: Layout.Renderer = (props) => {
  const mode = Base.useSelectMode();
  if (mode === "graph") return <GraphEditor {...props} />;
  return <TextEditor {...props} />;
};

export const Editor: Layout.Renderer = (props) => (
  <Base.Suspended arcKey={props.layoutKey}>
    <Internal {...props} />
  </Base.Suspended>
);

Editor.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
