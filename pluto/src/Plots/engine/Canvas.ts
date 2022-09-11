export type PlottingCanvasProps = {
  canvas?: HTMLCanvasElement;
  antialias?: boolean;
  resize?: boolean;
};

/** Canvas instantiates and wraps a document level overlay canvas element that can be used to render pluto plots
 * intermixed with other DOM elements.
 */
export default class Canvas {
  canvas: HTMLCanvasElement;
  webgl: WebGL2RenderingContext;
  canvasObserver?: ResizeObserver;

  /**
   * Construct a new Canvas given the specified props.
   * @param props - The props to use when constructing the Canvas.
   * @property props.canvas - The canvas element to use for rendering. If not specified, a new canvas element will be
   * created and appended to the document body.
   * @property props.antialias - Whether to enable antialiasing. Defaults to true.
   * @property props.resize - Whether to enable automatic resizing of the canvas to match its CSS display size. Defaults to true.
   * @returns A new Canvas.
   */
  constructor(props: PlottingCanvasProps) {
    this.canvas = this.initializeCanvas(props);
    this.webgl = this.initializeWebGL(props);
    this.canvasObserver = this.initializeResize(props);
  }

  /** resizeCanvasToDisplaySize resizes the canvas to match its CSS display size.
   **/
  resizeToDisplaySize() {
    const displayWidth = this.canvas.clientWidth;
    const displayHeight = this.canvas.clientHeight;
    const needResize =
      this.canvas.width !== displayWidth ||
      this.canvas.height !== displayHeight;
    if (needResize) {
      this.canvas.width = this.canvas.clientWidth;
      this.canvas.height = this.canvas.clientHeight;
    }
    return needResize;
  }

  /** close the Canvas and removes it from the DOM, freeing all of its resources in the process.
   */
  close() {
    if (this.canvasObserver) {
      this.canvasObserver.disconnect();
    }
    this.canvas.remove();
  }

  private initializeCanvas(props: PlottingCanvasProps): HTMLCanvasElement {
    if (props.canvas) return props.canvas;
    const canvas = document.createElement("canvas");
    canvas.id = "pluto-plot-canvas";
    return canvas;
  }

  private initializeWebGL({
    antialias,
  }: PlottingCanvasProps): WebGL2RenderingContext {
    const webgl = this.canvas.getContext("webgl2", {
      antialias,
    }) as WebGL2RenderingContext;
    if (!webgl) {
      throw new Error("[Pluto] - Canvas could not initialize WebGL2 context");
    }
    webgl.clearColor(1, 0, 0, 0);
    webgl.clear(webgl.COLOR_BUFFER_BIT);
    webgl.enable(webgl.BLEND);
    webgl.blendFunc(webgl.SRC_ALPHA, webgl.ONE_MINUS_SRC_ALPHA);
    return webgl;
  }

  private clear() {
    this.webgl.clearColor(1, 0, 0, 0);
    this.webgl.clear(this.webgl.COLOR_BUFFER_BIT);
  }

  private initializeResize(
    props: PlottingCanvasProps
  ): ResizeObserver | undefined {
    this.resizeToDisplaySize();
    if (!props.resize) return undefined;
    const observer = new ResizeObserver(() => this.resizeToDisplaySize());
    observer.observe(this.canvas);
    return observer;
  }
}
