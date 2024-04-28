import { aether } from "@/ether";
import { z } from "zod";
import { synnax } from "@/synnax/aether";
import { render } from "@/vis/render";
import { ranger, signals } from "@synnaxlabs/client";
import { TimeRange, bounds, box, change, scale, xy } from "@synnaxlabs/x";
import { Draw2D } from "@/vis/draw2d";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";

const hoveredStateZ = z.object({
  rangeKey: z.string(),
  timeRange: TimeRange.z,
  viewport: bounds.bounds,
});

type HoveredState = z.infer<typeof hoveredStateZ>;

export const providerStateZ = z.object({
  cursor: xy.xy.or(z.null()),
  hovered: hoveredStateZ.or(z.null()),
});

interface InternalState {
  ranges: Map<string, ranger.Range>;
  render: render.Context;
  draw: Draw2D;
  tracker: signals.Observable<string, ranger.Range>;
}

interface AnnotationProps {
  dataToDecimalScale: scale.Scale;
  viewport: box.Box;
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

    if (this.internal.tracker != null) return;
    this.internal.tracker = await client.ranges.openTracker();
    this.internal.tracker.onChange(async (c) => {
      c.forEach(async (r) => {
        if (r.variant === "delete") this.internal.ranges.delete(r.key);
        else this.internal.ranges.set(r.key, r.value);
      });
      render.Controller.requestRender(this.ctx, render.REASON_TOOL);
    });
  }

  async render(props: AnnotationProps): Promise<void> {
    const { dataToDecimalScale, region, viewport } = props;
    const { draw, render } = this.internal;
    const regionScale = dataToDecimalScale.scale(box.xBounds(region));
    const cursor = this.state.cursor == null ? null : this.state.cursor.x;
    let hoveredState: HoveredState | null = null;
    this.internal.ranges.forEach((r) => {
      const startPos = regionScale.pos(Number(r.timeRange.start.valueOf()));
      const endPos = regionScale.pos(Number(r.timeRange.end.valueOf()));
      const c = new color.Color(r.color as string);
      let hovered = false;
      if (cursor != null)
        hovered = bounds.contains({ lower: startPos, upper: endPos }, cursor);
      if (hovered)
        hoveredState = {
          rangeKey: r.key,
          timeRange: r.timeRange,
          viewport: {
            lower: dataToDecimalScale
              .scale(box.xBounds(viewport))
              .pos(Number(r.timeRange.start.valueOf())),
            upper: dataToDecimalScale
              .scale(box.xBounds(viewport))
              .pos(Number(r.timeRange.end.valueOf())),
          },
        };
      draw.container({
        region: box.construct(
          { x: startPos, y: box.top(region) - 1 },
          { x: endPos, y: box.bottom(region) - 1 },
        ),
        backgroundColor: c.setAlpha(0.1),
        bordered: false,
      });
      const titleRegion = box.construct(
        { x: startPos, y: box.top(region) - 34 },
        { x: endPos - 1, y: box.top(region) - 12 },
      );
      draw.container({
        region: titleRegion,
        backgroundColor: (t) => (hovered ? t.colors.gray.l2 : t.colors.gray.l1),
        bordered: true,
        borderWidth: 1,
        borderRadius: 6,
        borderColor: c,
      });
      draw.text({
        text: r.name,
        position: { x: startPos + 8, y: box.top(region) - 30 },
        level: "small",
        shade: 7,
      });
    });
    if (hoveredState != null && !this.state.hovered)
      this.setState((s) => ({ ...s, hovered: hoveredState }));
    if (hoveredState == null && this.state.hovered)
      this.setState((s) => ({ ...s, hovered: null }));
  }
}
