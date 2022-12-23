const canvas = document.createElement("canvas");

export const textWidth = (text: string, font: string): number => {
  const context = canvas.getContext("2d") as CanvasRenderingContext2D;
  context.font = font;
  const metrics = context.measureText(text);
  return Math.trunc(metrics.width);
};
