import { box, xy } from "@synnaxlabs/x";
import { type Box } from "@synnaxlabs/x/dist/spatial/box";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/aetherIndex";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const tableStateZ = z.object({
  region: box.box,
  widths: z.array(z.number()),
});

interface TableChildRenderProps {
  pos: xy.XY;
  widths: number[];
}

interface TableChild extends aether.Component {
  height: () => Promise<number>;
  render: (props: TableChildRenderProps) => Promise<void>;
}

interface InternalState {
  renderCtx: render.Context;
}

export class Table extends aether.Composite<
  typeof tableStateZ,
  InternalState,
  TableChild
> {
  static readonly TYPE = "Table";
  schema = tableStateZ;
  private readonly erase: render.Eraser = new render.Eraser();

  afterUpdate(): void {
    this.internal.renderCtx = render.Context.use(this.ctx);
    this.internal.renderCtx.loop.set({
      key: this.key,
      render: async () => await this.render(),
      canvases: ["upper2d"],
      priority: "high",
    });
  }

  async render(): Promise<render.Cleanup> {
    let accumulatedHeight = box.top(this.state.region);
    for (const c of this.children) {
      await c.render({
        pos: {
          x: box.left(this.state.region),
          y: accumulatedHeight,
        },
        widths: this.state.widths,
      });
      accumulatedHeight += await c.height();
    }
    return async ({ canvases }) => {
      this.erase.erase(
        this.internal.renderCtx,
        this.state.region,
        this.prevState?.region,
        xy.construct(10),
        canvases,
      );
    };
  }
}

export const trStateZ = z.object({
  height: z.number(),
});

interface TDRenderProps {
  box: Box;
}

export interface TD extends aether.Component {
  render: (props: TDRenderProps) => Promise<void>;
}

export class TR extends aether.Composite<typeof trStateZ, InternalState, TD> {
  static readonly TYPE = "TR";
  schema = trStateZ;

  async render({ pos, widths }: TableChildRenderProps): Promise<void> {
    const height = this.state.height;
    let accumulatedWidth = pos.x;
    const i = 0;
    for (const c of this.children) {
      await c.render({
        box: box.construct(accumulatedWidth, pos.y, widths[i], height),
      });
      accumulatedWidth += widths[i];
    }
  }

  async height(): Promise<number> {
    return this.state.height;
  }
}

export const stringTDStateZ = z.object({
  stringSource: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  level: z.string().optional().default("small"),
  color: color.Color.z.optional().default(color.ZERO),
});

interface InternalState {
  stringSource: telem.StringSource;
  draw: Draw2D;
}

export class StringTD
  extends aether.Leaf<typeof stringTDStateZ, InternalState>
  implements TD
{
  static readonly TYPE = "StringTD";
  schema = stringTDStateZ;

  afterUpdate(): void {
    const { internal: i } = this;
    i.stringSource = telem.useSource(this.ctx, this.state.stringSource, i.stringSource);
    i.draw = new Draw2D(render.Context.use(this.ctx).upper2d, theming.use(this.ctx));
  }

  async render({ box: b }: TDRenderProps): Promise<void> {
    const value = await this.internal.stringSource.value();
    this.internal.draw.drawTextInCenter({
      text: value,
      level: "small",
      box: b,
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Table.TYPE]: Table,
  [TR.TYPE]: TR,
  [StringTD.TYPE]: StringTD,
};
