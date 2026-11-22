// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { Canvas } from "@synnaxlabs/pluto";
import type { PropsWithChildren, ReactNode } from "react";

export interface GPUBoundaryProps extends PropsWithChildren {
  /** The plural name of the visualization, such as "Line plots". */
  name: string;
}

/** Renders its children only when the canvas has a GPU to draw them with. */
export const GPUBoundary = ({ name, children }: GPUBoundaryProps): ReactNode => {
  if (!Canvas.useGPUMissing()) return children;
  return (
    <Status.Summary
      center
      variant="error"
      description={`${name} need a GPU to render. Run the Console on a machine with a GPU.`}
    >
      GPU required
    </Status.Summary>
  );
};
