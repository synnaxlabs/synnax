// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { type ScannedNode } from "@/feature/opcua/task/types";

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

const ArrayVariableIcon = Icon.createFromPath(
  "variable-array",
  "M5,3h3c0.55,0 1,0.45 1,1c0,0.55 -0.45,1 -1,1h-2v14h2c0.55,0 1,0.45 1,1c0,0.55 -0.45,1 -1,1h-3c-0.55,0 -1,-0.45 -1,-1v-16c0,-0.55 0.45,-1 1,-1zM18,5h-2c-0.55,0 -1,-0.45 -1,-1c0,-0.55 0.45,-1 1,-1h3c0.55,0 1,0.45 1,1v16c0,0.55 -0.45,1 -1,1h-3c-0.55,0 -1,-0.45 -1,-1c0,-0.55 0.45,-1 1,-1h2zM10,10h-1c-0.55,0 -1,-0.45 -1,-1c0,-0.55 0.45,-1 1,-1h1c0.08,0 0.15,0.01 0.22,0.02c0.36,0.06 0.77,0.2 1.02,0.41c0.27,0.23 0.52,0.63 0.67,0.97c0.14,0.32 0.28,0.72 0.44,1.17c0.06,0.17 0.16,0.44 0.26,0.71c0.43,-0.61 0.94,-1.27 1.49,-1.81c0.38,-0.38 0.84,-0.76 1.28,-1.01c0.45,-0.26 1.1,-0.33 1.12,-0.34c0.15,-0.08 0.32,-0.13 0.5,-0.13c0.55,0 1,0.45 1,1c0,0.55 -0.45,1 -1,1c-0.02,0 -0.03,0 -0.05,0c-0.22,0.06 -0.45,0.13 -0.57,0.2c-0.29,0.17 -0.55,0.38 -0.86,0.69c-0.63,0.63 -1.19,1.41 -1.68,2.15c-0.01,0.01 -0.01,0.02 -0.02,0.02c-0.11,0.16 -0.23,0.34 -0.35,0.51c0.03,0.08 0.05,0.15 0.07,0.2c0.15,0.44 0.26,0.76 0.37,1c0.04,0.09 0.06,0.16 0.07,0.21c0.01,0 0.01,0 0.02,0h1c0.55,0 1,0.45 1,1c0,0.55 -0.45,1 -1,1h-1c-0.08,0 -0.15,-0.01 -0.22,-0.02c-0.36,-0.06 -0.77,-0.2 -1.02,-0.41c-0.27,-0.23 -0.52,-0.63 -0.67,-0.97c-0.03,-0.06 -0.05,-0.12 -0.08,-0.19c-0.04,0.04 -0.07,0.08 -0.11,0.11c-0.38,0.38 -0.84,0.76 -1.28,1.01c-0.45,0.26 -1.1,0.33 -1.12,0.34c-0.15,0.08 -0.32,0.13 -0.5,0.13c-0.55,0 -1,-0.45 -1,-1c0,-0.55 0.45,-1 1,-1c0.02,0 0.03,0 0.05,0c0.22,-0.06 0.45,-0.13 0.57,-0.2c0.29,-0.17 0.55,-0.38 0.86,-0.69c0.26,-0.26 0.51,-0.54 0.74,-0.84c-0.06,-0.16 -0.11,-0.28 -0.13,-0.35c-0.01,-0.02 -0.02,-0.04 -0.02,-0.06c-0.04,-0.11 -0.45,-1.18 -0.6,-1.61c-0.16,-0.45 -0.27,-0.77 -0.38,-1.02c-0.04,-0.1 -0.06,-0.17 -0.07,-0.22c-0.01,0 -0.01,0 -0.02,0z",
);

/** Returns the icon of an OPC UA node, which follows its node class. */
export const resolveIcon = (node: ScannedNode): ReactElement | undefined =>
  node.isArray ? <ArrayVariableIcon /> : ICONS[node.nodeClass];

export interface HaulPreviewProps {
  node: ScannedNode;
  /** How many nodes the drag carries. */
  count?: number;
}

/** The drag preview of an OPC UA node: its name, data type, and node ID. */
export const HaulPreview = ({ node, count }: HaulPreviewProps): ReactElement => {
  const facts = [node.dataType, node.nodeId].filter((fact) => fact !== "");
  return (
    <Haul.Preview
      icon={resolveIcon(node)}
      name={node.name}
      facts={facts}
      count={count}
    />
  );
};
