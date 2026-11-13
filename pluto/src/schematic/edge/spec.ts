// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type FC } from "react";

import { type ColorFallbacks, type FormProps } from "@/schematic/node/spec";
import { type Diagram } from "@/vis/diagram";

export interface EdgeProps extends Diagram.EdgeProps {
  onSegmentsChange: (segments: schematic.Segment[]) => void;
  edge: schematic.Edge;
}

export type Edge = FC<EdgeProps>;

export interface Spec {
  key: schematic.EdgeVariant;
  name: string;
  Form: FC<FormProps>;
  Edge: Edge;
  /** Returns the full width the edge draws at a stroke width, extra lines included. */
  renderedWidth: (strokeWidth: number) => number;
  /** Overrides the default fallback of each color field it lists. */
  colorFallbacks?: ColorFallbacks;
}
