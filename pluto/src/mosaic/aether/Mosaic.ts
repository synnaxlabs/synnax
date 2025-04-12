import { box, direction } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/ether";

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

const nodeZ: z.ZodType<BaseNode> = baseNodeZ.extend({
  first: z.lazy(() => nodeZ).optional(),
  last: z.lazy(() => nodeZ).optional(),
});

export const mosaicStateZ = z.object({
  root: nodeZ,
  box: box.box,
});

export interface ContextValue {
  resolveBox: (tabKey: string) => box.Box;
}

const CONTEXT_KEY = "mosaic.context";

export class Mosaic extends aether.Composite<typeof mosaicStateZ, {}, {}> {
  static readonly TYPE = "Mosaic";
  static readonly stateZ = mosaicStateZ;
  schema = Mosaic.stateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    const sizes = calculateSizes(this.state.root, this.state.box);
    ctx.set(CONTEXT_KEY, {
      resolveBox: (tabKey: string) => sizes[tabKey],
    });
  }
}

export const useGetBox = (ctx: aether.Context, tabKey: string): (() => box.Box) => {
  const value: ContextValue = ctx.get(CONTEXT_KEY);
  return () => value.resolveBox(tabKey);
};

export const REGISTRY: aether.ComponentRegistry = {
  [Mosaic.TYPE]: Mosaic,
};

export const calculateSizes = (
  root: BaseNode,
  bbx: box.Box,
): Record<string, box.Box> => {
  const sizes: Record<string, box.Box> = {};
  const TAB_BAR_HEIGHT = 27;

  if (root.first == null && root.last == null) {
    if (root.selected == null) return sizes;
    // For leaf nodes with tabs, adjust the box to account for tab bar
    sizes[root.selected] = box.construct(
      box.x(bbx),
      box.y(bbx) + TAB_BAR_HEIGHT,
      box.width(bbx),
      box.height(bbx) - TAB_BAR_HEIGHT,
    );
    return sizes;
  }

  const splitDirection = root.direction ?? "x";
  const ratio = root.size ?? 0.5;

  if (splitDirection === "x") {
    const firstWidth = box.width(bbx) * ratio;
    const lastWidth = box.width(bbx) * (1 - ratio);

    if (root.first) {
      const firstBox = box.construct(
        box.x(bbx),
        box.y(bbx),
        firstWidth,
        box.height(bbx),
      );
      Object.assign(sizes, calculateSizes(root.first, firstBox));
    }

    if (root.last) {
      const lastBox = box.construct(
        box.x(bbx) + firstWidth,
        box.y(bbx),
        lastWidth,
        box.height(bbx),
      );
      Object.assign(sizes, calculateSizes(root.last, lastBox));
    }
  } else {
    const firstHeight = box.height(bbx) * ratio;
    const lastHeight = box.height(bbx) * (1 - ratio);

    if (root.first) {
      const firstBox = box.construct(
        box.x(bbx),
        box.y(bbx),
        box.width(bbx),
        firstHeight,
      );
      Object.assign(sizes, calculateSizes(root.first, firstBox));
    }

    if (root.last) {
      const lastBox = box.construct(
        box.x(bbx),
        box.y(bbx) + firstHeight,
        box.width(bbx),
        lastHeight,
      );
      Object.assign(sizes, calculateSizes(root.last, lastBox));
    }
  }

  return sizes;
};
