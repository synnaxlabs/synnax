import { type Destructor } from "@synnaxlabs/x";
import { type Box } from "@synnaxlabs/x/dist/spatial/box";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/core";
import { noop } from "@/telem/noop";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const tableStateZ = z.object({});

interface InternalState {}

interface TableChild extends aether.Component {
  height: () => Promise<number>;
}

export class Table extends aether.Composite<typeof tableStateZ, TableChild> {
  static readonly TYPE = "Table";
  schema = tableStateZ;
}

export const trStateZ = z.object({
  height: z.number(),
});

interface TDRenderProps {
  box: Box;
}

export interface TD extends aether.Component {
  width: () => Promise<number>;
  render: (props: TDRenderProps) => Promise<void>;
}

export class TR extends aether.Composite<typeof trStateZ, TD> {}

export const stringTDZ = z.object({
  stringSource: telem.stringSpecZ.optional().default(noop.stringSourceSpec),
});

interface InternalState {
  stringSource: telem.StringSource;
  cleanupStringSource: Destructor;
  draw: Draw2D;
}

export class StringTD
  extends aether.Leaf<typeof stringTDZ, InternalState>
  implements TD
{
  afterUpdate(): void {
    [this.internal.stringSource, this.internal.cleanupStringSource] =
      telem.use<telem.StringSource>(this.ctx, this.key, this.state.stringSource);
    this.internal.draw = new Draw2D(
      render.Context.use(this.ctx).upper2d,
      theming.use(this.ctx),
    );
  }

  async width(): Promise<number> {
    return 0;
  }

  async render({ box }: TDRenderProps): Promise<void> {
    const value = await this.internal.stringSource.string();
    this.internal.draw.drawTextInCenter({
      text: value,
      box,
      level: "p",
    });
  }
}
