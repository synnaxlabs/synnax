import { color, type destructor, xy } from "@synnaxlabs/x";
import { type Context } from "./context";
/** A general purpose compiler and utility container for workign with WebGL programs. */
export declare class GLProgram {
    /** The render context used by this program. */
    readonly renderCtx: Context;
    /** The underlying webgl program. */
    readonly prog: WebGLProgram;
    /** The code for the vertex shader. */
    private readonly vertShader;
    /** The code for the fragment shader. */
    private readonly fragShader;
    uniformLocCache: Map<string, WebGLUniformLocation>;
    /**
     * @constructor compiles the given vertex and fragment shaders under the given render
     * context into a program.
     * @param ctx - The render context to use.
     * @param vertShader - The vertex shader code.
     * @param fragShader - The fragment shader code.
     */
    constructor(ctx: Context, vertShader: string, fragShader: string);
    /** Sets the current program as the active program used by the context. */
    setAsActive(): destructor.Destructor;
    /** Sets a uniform XY coordinate value. */
    uniformXY(name: string, value: xy.Crude): void;
    /** Sets a uniform color value. */
    uniformColor(name: string, value: color.Color): void;
    private getUniformLoc;
    private compile;
    private compileShader;
}
//# sourceMappingURL=GLProgram.d.ts.map