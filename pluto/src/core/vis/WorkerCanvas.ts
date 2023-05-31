import { Box } from "@synnaxlabs/x";
import { z } from "zod";

import { WComposite } from "../bob/worker";

import { LinePlot, LinePlotFactory } from "./LinePlot/worker";
import { RenderContext, RenderQueue } from "./render";
import { TelemProvider } from "./telem/TelemService";

export const canvasState = z.object({
  dpr: z.number(),
  region: Box.z,
});

export type CanvasState = z.input<typeof canvasState>;
export type ParsedCanvasState = z.output<typeof canvasState>;

export class Canvas extends WComposite<LinePlot, CanvasState, ParsedCanvasState> {
  ctx: RenderContext;
  queue: RenderQueue;
  telem: TelemProvider;

  static readonly TYPE = "canvas";

  constructor(
    ctx: RenderContext,
    key: string,
    linePlotFactory: LinePlotFactory,
    telem: TelemProvider,
    props: CanvasState,
    queue: RenderQueue
  ) {
    super(Canvas.TYPE, key, linePlotFactory, canvasState, props);
    this.ctx = ctx;
    this.telem = telem;
    this.queue = queue;
    this.setHook(() => {
      this.ctx.resize(new Box(this.state.region), this.state.dpr);
    });
  }
}

export const bootstrap = canvasState.extend({
  key: z.string(),
  glCanvas: z.instanceof(OffscreenCanvas),
  canvasCanvas: z.instanceof(OffscreenCanvas),
});

export type Bootstrap = z.output<typeof bootstrap>;

export const newBootstrapFn = (telem: TelemProvider) => (props: Bootstrap) => {
  const box = new Box(props.region);
  const renderCtx = new RenderContext(
    props.glCanvas,
    props.canvasCanvas,
    box,
    props.dpr
  );
  const renderQueue = new RenderQueue();
  const linePlotFactory = new LinePlotFactory(renderCtx, renderQueue, telem);
  return new Canvas(renderCtx, props.key, linePlotFactory, telem, props, renderQueue);
};
