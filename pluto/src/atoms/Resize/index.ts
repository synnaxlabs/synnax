import { Resize as CoreResize } from "./Resize";
import {
	ResizeMultiple,
	useResizeMultiple,
	UseResizeMultipleProps,
} from "./ResizeMultiple";

type CoreResizeType = typeof CoreResize;

interface ResizeType extends CoreResizeType {
	Multiple: typeof ResizeMultiple;
	useMultiple: typeof useResizeMultiple;
}

export const Resize = CoreResize as ResizeType;

Resize.Multiple = ResizeMultiple;
Resize.useMultiple = useResizeMultiple;
