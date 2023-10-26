// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, box, type scale } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/aetherIndex";
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const tableSourceZ = z.object({
  key: z.string(),
  valueSource: telem.stringSpecZ.optional().default(noop.stringSourceSpec),
});

export const tableStateZ = z.object({
  sources: z.array(tableSourceZ),
  region: box.box,
  size: z.number(),
  scroll: z.number().optional().default(0),
});

interface TableSourceInternalState {
  valueSource: telem.StringSource;
  cleanupValueSource: Destructor;
}

interface InternalState {
  sources: TableSourceInternalState[];
  render: render.Context;
  t: theming.Theme;
}

export class Table extends aether.Leaf<typeof tableStateZ, InternalState> {
  static readonly TYPE = "Table";
  schema = tableStateZ;

  afterUpdate(): void {
    this.internal.sources = [];
    this.state.sources.forEach((s) => {
      const [t, cleanup] = telem.use<telem.StringSource>(
        this.ctx,
        s.key,
        s.valueSource,
      );
      this.internal.sources.push({
        valueSource: t,
        cleanupValueSource: cleanup,
      });
    });
    this.internal.render = render.Context.use(this.ctx);
    this.internal.t = theming.use(this.ctx);
    render.Controller.requestRender(this.ctx, "table");
  }

  afterDelete(): void {
    this.internal.sources.forEach((s) => s.cleanupValueSource());
    render.Controller.requestRender(this.ctx, "table");
  }

  async render({ s }: { s?: scale.XY }): Promise<render.Cleanup> {
    if (this.deleted || s == null || box.isZero(this.state.region))
      return async () => {};
    const { sources, t } = this.internal;
    const draw = new Draw2D(this.internal.render.upper2d.applyScale(s), t);
    const text = await Promise.all(
      sources.map(async (s) => await s.valueSource.string()),
    );
    const [, d] = draw.spacedTextDrawF({
      text,
      direction: "y",
      spacing: 1,
      level: "small",
    });
    d(box.topLeft(this.state.region));
    return async () => {};
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Table.TYPE]: Table,
};
