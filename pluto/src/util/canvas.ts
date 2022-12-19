const context = document
  .createElement("canvas")
  .getContext("2d") as CanvasRenderingContext2D;

export const measureTextWidth = (text: string, font: string): number => {
  context.font = font;
  const metrics = context.measureText(text);
  return Math.trunc(metrics.width);
};
