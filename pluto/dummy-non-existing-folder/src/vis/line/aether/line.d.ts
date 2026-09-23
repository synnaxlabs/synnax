import { type Instrumentation } from "@synnaxlabs/alamos";
import { bounds, type box, color, DataType, type destructor, type MultiSeries, type scale, type Series, TimeSpan, xy } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { telem } from "../../../telem/aether";
import { render } from "../../render";
export declare const stateZ: z.ZodObject<{
    x: z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"series">;
    }, z.core.$strip>;
    y: z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"series">;
    }, z.core.$strip>;
    label: z.ZodOptional<z.ZodString>;
    color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>;
    strokeWidth: z.ZodDefault<z.ZodNumber>;
    downsample: z.ZodDefault<z.ZodNumber>;
    downsampleMode: z.ZodDefault<z.ZodEnum<{
        average: "average";
        decimate: "decimate";
    }>>;
    visible: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export type State = z.input<typeof stateZ>;
export type ParsedState = z.infer<typeof stateZ>;
export declare const DEFAULT_OVERLAP_THRESHOLD: TimeSpan;
export interface FindResult {
    key: string;
    position: xy.XY;
    value: xy.XY;
    color: color.Color;
    label?: string;
    units?: string;
    bounds: bounds.Bounds;
}
export declare const ZERO_FIND_RESULT: FindResult;
export interface LineProps {
    /**
     * A box in pixel space representing the region of the display that the line should be
     * rendered in. The root of the pixel coordinate system is the top left of the canvas.
     */
    region: box.Box;
    /** An XY scale that maps from the data space to decimal space. */
    dataToDecimalScale: scale.XY;
    exposure: number;
}
export declare class GLProgram extends render.GLProgram {
    private readonly translationBufferCache;
    constructor(ctx: render.Context, vertShader: string, fragShader: string);
    bindState({ strokeWidth, color }: ParsedState): number;
    bindScale(dataScaleTransform: scale.XYTransformT, regionTransform: scale.XYTransformT): void;
    draw({ x, y, count, downsample, xOffset, yOffset }: DrawOperation, instances: number, xDataType: DataType, yDataType: DataType): void;
    private bindAttrBuffer;
    private getAndBindTranslationBuffer;
    /**
     * We apply stroke width by drawing the line multiple times, each time with a slight
     * transformation. This is done as simply as possible. We draw the "centered" line and
     * then four more lines: one to the left, one to the right, one above, and one below.
     * We can repeat this process an arbitrary number of times to make the line thicker.
     * As we increase the stroke width, we also increase the cost of drawing the line.
     */
    private attrStrokeWidth;
}
export declare class Context {
    private static readonly CONTEXT_KEY;
    private readonly uint8HybridProgram;
    private readonly float32Program;
    private constructor();
    get gl(): WebGL2RenderingContext;
    getProgram(dataType: DataType): GLProgram;
    static create(ctx: aether.Context, renderCtx: render.Context): Context;
    static use(ctx: aether.Context): Context;
}
interface InternalState {
    instrumentation: Instrumentation;
    lineCtx: Context;
    xTelem: telem.SeriesSource;
    stopListeningXTelem?: destructor.Destructor;
    yTelem: telem.SeriesSource;
    stopListeningYTelem?: destructor.Destructor;
    requestRender: render.Requestor;
    xDownsampler: telem.SeriesDownsampler;
    yDownsampler: telem.SeriesDownsampler;
}
export declare class Line extends aether.Leaf<typeof stateZ, InternalState> {
    static readonly TYPE = "line";
    schema: typeof stateZ;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    get loading(): boolean;
    xBounds(): bounds.Bounds;
    /**
     * @param xWindow - the visible x range. Bounds cover only samples whose x value
     * falls inside it; when the window is non-finite or clips out every sample, the
     * source's full bounds are used instead.
     * @returns the y bounds of this line's samples inside the window.
     */
    yBounds(xWindow: bounds.Bounds): bounds.Bounds;
    findByXValue(props: LineProps, target: number): FindResult;
    render(props: LineProps): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export interface DrawOperation {
    x: Series;
    y: Series;
    xOffset: number;
    yOffset: number;
    count: number;
    downsample: number;
}
export declare const buildDrawOperations: (xSeries: MultiSeries, ySeries: MultiSeries, exposure: number, userSpecifiedDownSampling: number, downsampleMode: telem.DownsampleMode, overlapThreshold: TimeSpan) => DrawOperation[];
export {};
//# sourceMappingURL=line.d.ts.map