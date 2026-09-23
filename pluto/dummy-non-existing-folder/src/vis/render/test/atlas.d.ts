import { xy } from "@synnaxlabs/x";
import { SugaredOffscreenCanvasRenderingContext2D } from "../../draw2d/canvas";
import { type render } from "..";
/** Advance the atlas lays consecutive glyphs out on. */
export declare const ATLAS_ADVANCE: number;
/** Ink height of a glyph the atlas draws, all of it above the baseline. */
export declare const ATLAS_INK_HEIGHT = 8;
/** Distance from the top of the cell the atlas copies out to its baseline. */
export declare const ATLAS_BASELINE_OFFSET: number;
export interface AtlasSurface {
    /** Canvas that draws text through a real atlas onto the recording surface. */
    canvas: SugaredOffscreenCanvasRenderingContext2D;
    /** A `render.Context`-shaped value whose canvases are all {@link canvas}. */
    context: render.Context;
    /** Where the atlas copied each glyph out to, in draw order. */
    glyphs: () => xy.XY[];
    /** Drops the glyphs recorded so far. */
    clear: () => void;
}
/**
 * Construct a surface that runs the production text path: a real Sugared context over a
 * real atlas, recording where every glyph lands. Use it to assert where text was drawn,
 * which the {@link Recorder} cannot show, since it stands in for the atlas instead of
 * feeding it. Stubs `OffscreenCanvas`, so the caller must call `vi.unstubAllGlobals`
 * after the test.
 */
export declare const atlasSurface: () => AtlasSurface;
//# sourceMappingURL=atlas.d.ts.map