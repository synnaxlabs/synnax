// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Profiler, type ReactElement, type ReactNode } from "react";

export interface CountedProps {
  id: string;
  children: ReactNode;
}

export interface RenderCounter {
  /** Counts each commit that re-renders its children under id. Mounts do not count. */
  Counted: (props: CountedProps) => ReactElement;
  /** The re-render counts by id since creation or the last reset. */
  counts: Map<string, number>;
  reset: () => void;
}

/** @returns a counter of re-renders for each id, such as one per list item. */
export const createRenderCounter = (): RenderCounter => {
  const counts = new Map<string, number>();
  const Counted = ({ id, children }: CountedProps): ReactElement => (
    <Profiler
      id={id}
      onRender={(_, phase) => {
        if (phase !== "mount") counts.set(id, (counts.get(id) ?? 0) + 1);
      }}
    >
      {children}
    </Profiler>
  );
  return { Counted, counts, reset: () => counts.clear() };
};
