import { type Context } from "../context";
import { GLProgram } from "../GLProgram";
/**
 * You may be wondering, why does this program that draws a colorless, zero-footprint
 * triangle exist? It turns out that the WeBGL implementation on windows doesn't actually
 * clear a scissored region of the screen when you call `gl.clear`, you actually need
 * to make a draw call to replace what's currently in the framebuffer. This is a
 * workaround that does exactly that.
 */
export declare class Program extends GLProgram {
    private readonly positionBuffer;
    constructor(ctx: Context);
    exec(): void;
}
//# sourceMappingURL=program.d.ts.map