import { box, direction } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

const baseNodeZ = z.object({
  key: z.number(),
  selected: z.string().optional(),
  direction: direction.direction.optional(),
  size: z.number().optional(),
});

type BaseNode = z.infer<typeof baseNodeZ> & {
  first?: BaseNode;
  last?: BaseNode;
};

export const nodeZ: z.ZodType<BaseNode> = baseNodeZ.extend({
  first: z.lazy(() => nodeZ).optional(),
  last: z.lazy(() => nodeZ).optional(),
});

export const mosaicStateZ = z.object({
  box: box.box,
  root: nodeZ,
});

const CONTEXT_KEY = "mosaic";

export class Mosaic extends aether.Composite<typeof mosaicStateZ> {
  static readonly TYPE = "mosaic";
  schema: typeof mosaicStateZ = mosaicStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    ctx.set(CONTEXT_KEY, this.state);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Mosaic.TYPE]: Mosaic,
};
