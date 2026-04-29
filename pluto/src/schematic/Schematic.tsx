// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { TimeSpan } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  "dragHandleSelector"
> {}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = (renderers: Diagram.RendererConfig): FC<SchematicProps> => {
  const Base = Diagram.create(renderers);
  const Schematic = ({ className, ...props }: SchematicProps): ReactElement => (
    <Base
      className={CSS(CSS.B("schematic"), className)}
      dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      {...props}
    />
  );
  return Schematic;
};
