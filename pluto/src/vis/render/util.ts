import { box, xy } from "@synnaxlabs/x";

export const applyOverScan = (b: box.Box, overScan: xy.XY): box.Box =>
  box.construct(
    box.left(b) - overScan.x,
    box.top(b) - overScan.y,
    box.width(b) + overScan.x * 2,
    box.height(b) + overScan.y * 2,
  );
