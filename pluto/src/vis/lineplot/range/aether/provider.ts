import { aether } from "@/ether";
import { z } from "zod";
import { synnax } from "@/synnax/aether";
import { render } from "@/vis/render";
import { ranger } from "@synnaxlabs/client";
import { box, scale, xy } from "@synnaxlabs/x";
import { Draw2D } from "@/vis/draw2d";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";

export const providerStateZ = z.object({});

interface InternalState {
  ranges: Map<string, ranger.Range>;
  render: render.Context;
  draw: Draw2D;
}

interface AnnotationProps {
  dataToDecimalScale: scale.Scale;
  region: box.Box;
}

export class Provider extends aether.Leaf<typeof providerStateZ, InternalState> {
  static readonly TYPE = "range-provider";
  schema = providerStateZ;

  async afterUpdate(): Promise<void> {
    this.internal.render = render.Context.use(this.ctx);
    this.internal.draw = new Draw2D(
      this.internal.render.upper2d,
      theming.use(this.ctx),
    );

    if (this.internal.ranges == null) this.internal.ranges = new Map();
    const client = synnax.use(this.ctx);
    if (client == null) return;

    const t = await client.ranges.openTracker();
    t.onChange(async (c) => {
      console.log("CHANGE", c);
      c.forEach(async (r) => {
        if (r.variant === "delete") this.internal.ranges.delete(r.key);
        else this.internal.ranges.set(r.key, r.value);
      });
      render.Controller.requestRender(this.ctx, render.REASON_TOOL);
    });
  }

  async render(props: AnnotationProps): Promise<void> {
    const { dataToDecimalScale, region } = props;
    const { draw, render } = this.internal;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    const clearScissor = render.scissor(region, xy.ZERO, ["upper2d"]);
    this.internal.ranges.forEach((r) => {
      const startPos = regionScale.pos(Number(r.timeRange.start.valueOf()));
      const endPos = regionScale.pos(Number(r.timeRange.end.valueOf()));
      const c = new color.Color("#2be29f");
      draw.container({
        region: box.construct(
          { x: startPos, y: box.top(region) - 1 },
          { x: endPos, y: box.bottom(region) - 1 },
        ),
        backgroundColor: c.setAlpha(0.07),
      });
      draw.rule({
        stroke: c,
        lineWidth: 1,
        lineDash: 0,
        direction: "y",
        region,
        position: startPos,
      });
      draw.rule({
        stroke: c,
        lineWidth: 1,
        lineDash: 0,
        direction: "y",
        region,
        position: endPos,
      });
      const titleRegion = box.construct(
        { x: startPos + 1, y: box.top(region) },
        { x: endPos - 1, y: box.top(region) + 20 },
      );
      draw.container({
        region: titleRegion,
        backgroundColor: c.setAlpha(0.1),
        bordered: "bottom",
        borderColor: c,
      });
      draw.text({
        text: `Annotation`,
        position: { x: startPos + 6, y: box.top(region) + 3 },
        level: "p",
      });
    });
    clearScissor();
  }
}
