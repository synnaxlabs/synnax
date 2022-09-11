import Canvas from "./Canvas";
import { combineTransforms, pixelsToClipSpace, XYTransform } from "./xy";
import { DOMElement } from "react";

type TransformablePlotter = {
  setTransform(transform: XYTransform): void;
  plot(): void;
};

type Margins = {
  top: number;
  right: number;
  bottom: number;
  left: number;
};

export default class Positioner {
  context: Canvas;
  domElement: Element;
  plotter: TransformablePlotter;
  margins: Margins;

  constructor(
    context: Canvas,
    domElement: Element,
    plotter: TransformablePlotter,
    margins: Margins
  ) {
    this.context = context;
    this.domElement = domElement;
    this.plotter = plotter;
    this.margins = margins;
    const resizeObserver = new ResizeObserver(this.onResize.bind(this));
    resizeObserver.observe(domElement);
    resizeObserver.observe(context.webgl.canvas);
    this.onResize([]);
  }

  private onResize(event: ResizeObserverEntry[]) {
    const rect = this.domElement.getBoundingClientRect();
    console.log(rect.x, rect.y, rect.width, rect.height);
    this.plotter.setTransform(
      combineTransforms(
        {
          scale: [
            rect.width - this.margins.left - this.margins.right,
            rect.height - this.margins.top - this.margins.bottom,
          ],
          translation: [rect.x + this.margins.left, rect.y + this.margins.top],
        },
        pixelsToClipSpace([
          this.context.webgl.canvas.width,
          this.context.webgl.canvas.height,
        ])
      )
    );
    this.plotter.plot();
  }
}
