import { GLLineRenderer, LINE_RENDERER_TYPE } from "./line/renderer";
import { GLRenderer } from "./renderer";

export class GLRendererRegistry {
  private readonly renderers: Record<string, GLRenderer<any>> = {};

  register<R>(renderer: GLRenderer<R>): void {
    this.renderers[renderer.type] = renderer;
  }

  get<R>(type: string): GLRenderer<R> {
    return this.renderers[type] as GLRenderer<R>;
  }

  compile(gl: WebGLRenderingContext): void {
    Object.values(this.renderers).forEach((r) => r.compile(gl));
  }
}

export type DefaultRenderers = typeof LINE_RENDERER_TYPE;

export const newDefaultRendererRegistry = (): GLRendererRegistry => {
  const registry = new GLRendererRegistry();
  registry.register(new GLLineRenderer());
  return registry;
};
