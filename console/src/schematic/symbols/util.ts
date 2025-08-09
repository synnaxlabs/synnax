import { box } from "@synnaxlabs/x";

export const parseSVGViewBox = (element: SVGSVGElement): box.Box => {
  const viewBox = element.viewBox.baseVal;
  if (viewBox)
    return box.construct(
      {
        x: viewBox.x,
        y: viewBox.y,
      },
      { width: viewBox.width, height: viewBox.height },
    );
  const width = parseFloat(element.getAttribute("width") || "100");
  const height = parseFloat(element.getAttribute("height") || "100");
  return box.construct({ x: 0, y: 0, width, height });
};
