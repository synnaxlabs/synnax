// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Canvas.css";

import { Canvas as Core } from "@synnaxlabs/pluto";
import { type PropsWithChildren, ReactNode } from "react";

export interface CanvasProps extends PropsWithChildren<{}> {
  enabled?: boolean;
}

export const Canvas = ({ children, enabled = true }: CanvasProps): ReactNode => {
  if (!enabled) return children;
  return <Core.Canvas className="console-vis__canvas">{children}</Core.Canvas>;
};
