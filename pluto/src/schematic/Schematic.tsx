// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { type ReactElement } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { ConnectionLine, Edge, type EdgeData } from "@/schematic/edge";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps
  extends Omit<Diagram.DiagramProps, "dragHandleSelector"> {}

const edgeRenderer = Component.renderProp(Edge);

export const Schematic = ({
  className,
  children,
  ...props
}: SchematicProps): ReactElement => (
  <Diagram.Diagram
    className={CSS(CSS.B("schematic"), className)}
    dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
    {...props}
  >
    <Diagram.EdgeRenderer<EdgeData> connectionLineComponent={ConnectionLine}>
      {edgeRenderer}
    </Diagram.EdgeRenderer>
    {children}
  </Diagram.Diagram>
);
