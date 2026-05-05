// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { useUpdateNodeInternals } from "@xyflow/react";
import { type PropsWithChildren, type ReactElement, useEffect, useRef } from "react";

export interface BoundaryProps extends PropsWithChildren<{}> {
  orientation: location.Outer;
  refreshDeps?: unknown;
}

export const Boundary = ({
  children,
  orientation,
  refreshDeps,
}: BoundaryProps): ReactElement | null => {
  let updateInternals: ReturnType<typeof useUpdateNodeInternals> | undefined;
  try {
    updateInternals = useUpdateNodeInternals();
  } catch {
    return null;
  }
  const ref = useRef<HTMLDivElement & HTMLButtonElement>(null);
  const first = useRef<boolean>(true);
  useEffect(() => {
    if (ref.current == null) return;
    if (first.current) {
      first.current = false;
      return;
    }
    const node = ref.current.closest(".react-flow__node");
    const id = node?.getAttribute("data-id");
    if (id == null) return;
    updateInternals?.(id);
  }, [orientation, refreshDeps]);
  return (
    <>
      <span ref={ref} />
      {children}
    </>
  );
};
