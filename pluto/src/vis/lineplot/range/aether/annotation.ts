// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";
import { TimeStamp, box, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const annotationStateZ = z.object({
  start: TimeStamp.z,
  end: TimeStamp.z,
});

interface AnnotationProps {
  dataToDecimalScale: scale.Scale;
  region: box.Box;
}

interface InternalState {
  render: render.Context;
  draw: Draw2D;
}

export class Annotation extends aether.Leaf<typeof annotationStateZ, InternalState> {
  static readonly TYPE = "range-annotation";
  schema = annotationStateZ;

  async afterUpdate(): Promise<void> {
    this.internal.render = render.Context.use(this.ctx);
    this.internal.draw = new Draw2D(
      this.internal.render.upper2d,
      theming.use(this.ctx),
    );
  }

  async render(props: AnnotationProps): Promise<void> {
    const { dataToDecimalScale, region } = props;
    const { start, end } = this.state;
    const { draw, render } = this.internal;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
  }
}
