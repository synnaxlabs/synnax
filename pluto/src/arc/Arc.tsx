// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/Arc.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Diagram } from "@/vis/diagram";

export interface ArcProps extends Diagram.DiagramProps {}

const FIT_VIEW_OPTIONS: Diagram.FitViewOptions = {
  minZoom: 0.5,
  maxZoom: 0.9,
  padding: 0.1,
};

export const Arc = ({ className, ...props }: ArcProps): ReactElement => (
  <Diagram.Diagram
    className={CSS(className, CSS.B("arc"))}
    fitViewOptions={FIT_VIEW_OPTIONS}
    snapGrid={[2, 2]}
    snapToGrid
    {...props}
  />
);
