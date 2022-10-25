import { Resize as CoreResize } from "./Resize";
import { ResizeMultiple } from "./ResizeMultiple";

type CoreResizeType = typeof CoreResize;

interface ResizeType extends CoreResizeType {
  Multiple: typeof ResizeMultiple;
}

export const Resize = CoreResize as ResizeType;

Resize.Multiple = ResizeMultiple;
