// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { useNodeId, useUpdateNodeInternals } from "@xyflow/react";
import { type PropsWithChildren, type ReactElement, useEffect, useRef } from "react";

export interface BoundaryProps extends PropsWithChildren<{}> {
  orientation: location.Outer;
  refreshDeps?: unknown;
}

export const Boundary = ({
  orientation,
  refreshDeps,
  children,
}: BoundaryProps): ReactElement | null => {
  // Outside a React Flow node, as in a symbol preview, the handles have no node.
  const nodeID = useNodeId();
  if (nodeID == null) return null;
  return (
    <Refresh nodeID={nodeID} orientation={orientation} refreshDeps={refreshDeps}>
      {children}
    </Refresh>
  );
};

interface RefreshProps extends BoundaryProps {
  nodeID: string;
}

const Refresh = ({
  nodeID,
  orientation,
  refreshDeps,
  children,
}: RefreshProps): ReactElement => {
  const updateInternals = useUpdateNodeInternals();
  const first = useRef(true);
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    updateInternals(nodeID);
  }, [orientation, refreshDeps]);
  return <>{children}</>;
};
