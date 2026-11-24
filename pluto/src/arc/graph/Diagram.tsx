// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import type { record } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Node } from "@/arc/graph/node";
import { useNodeConfig, useSingleDispatch } from "@/arc/queries";
import { Diagram as Base } from "@/vis/diagram";

const NodeRenderer = ({
  nodeKey,
  position,
  selected,
  draggable,
}: Base.NodeProps): ReactElement | null => {
  const config = useNodeConfig({ nodeKey });
  const dispatch = useSingleDispatch();
  const handleChange = (config: Partial<record.Unknown>) =>
    dispatch(arc.setNodeInputs({ key: nodeKey, inputs: config }));
  if (config == null) return null;
  const Render = Node.resolveSpec(config.type).Symbol;
  return (
    <Render
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      config={config}
      onConfigChange={handleChange}
    />
  );
};

export const Diagram = Base.create({ node: Component.renderProp(NodeRenderer) });
