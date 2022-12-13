const canvas = document.createElement("canvas");

export const getTextWidth = (text: string, font: string) => {
  const context = canvas.getContext("2d") as CanvasRenderingContext2D;
  context.font = font;
  const metrics = context.measureText(text);
  return metrics.width;
};
