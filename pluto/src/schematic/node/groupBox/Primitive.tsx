// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/groupBox/groupBox.css";

import { useStore } from "@xyflow/react";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { PADDING } from "@/schematic/node/groupBox/config";

export interface PrimitiveProps {
  nodeKey: string;
  members: string[];
  className?: string;
}

interface Size {
  width: number;
  height: number;
}

/**
 * Primitive renders the group's box, sized from the members' rendered bounds
 * rather than stored state.
 */
export const Primitive = ({
  nodeKey,
  members,
  className,
}: PrimitiveProps): ReactElement => {
  const size = useStore(
    (s): Size => {
      const self = s.nodeLookup.get(nodeKey);
      let maxX = -Infinity;
      let maxY = -Infinity;
      for (const m of members) {
        const node = s.nodeLookup.get(m);
        if (node == null) continue;
        const { x, y } = node.internals.positionAbsolute;
        maxX = Math.max(maxX, x + (node.measured.width ?? 0));
        maxY = Math.max(maxY, y + (node.measured.height ?? 0));
      }
      if (self == null || maxX === -Infinity)
        return { width: 2 * PADDING, height: 2 * PADDING };
      const { x, y } = self.internals.positionAbsolute;
      return { width: maxX - x + PADDING, height: maxY - y + PADDING };
    },
    (a, b) => a.width === b.width && a.height === b.height,
  );
  return (
    <div
      className={CSS.cls(className, CSS.B("group-box"))}
      style={{ width: size.width, height: size.height }}
    />
  );
};
