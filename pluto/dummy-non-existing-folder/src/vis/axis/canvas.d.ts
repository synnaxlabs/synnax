import { color, dimensions, type location, xy } from "@synnaxlabs/x";
import { type Axis, type AxisProps, type AxisState, type ParsedAxisState, type RenderResult } from "./axis";
import { type Tick, type TickFactory } from "./ticks";
import { type render } from "../render";
export declare const newCanvas: (loc: location.Outer, ctx: render.Context, state: ParsedAxisState) => Axis;
export declare class Base {
    renderCtx: render.Context;
    state: ParsedAxisState;
    tickFactory: TickFactory;
    constructor(ctx: render.Context, state: ParsedAxisState);
    setState(state: AxisState): void;
    protected drawLine(start: xy.XY, end: xy.XY): void;
    protected drawTicks(ticks: Tick[], f: (textDimensions: dimensions.Dimensions, tick: Tick) => void): dimensions.Dimensions;
    private static START_BOUND;
    protected maybeDrawGrid(size: number, ticks: Tick[], f: (tick: Tick) => [xy.XY, xy.XY]): void;
    protected setColor(colorValue: color.Color): void;
}
export declare class Bottom extends Base implements Axis {
    render(props: AxisProps): RenderResult;
}
export declare class Top extends Base implements Axis {
    render(props: AxisProps): RenderResult;
}
export declare class Right extends Base implements Axis {
    render(props: AxisProps): RenderResult;
}
//# sourceMappingURL=canvas.d.ts.map