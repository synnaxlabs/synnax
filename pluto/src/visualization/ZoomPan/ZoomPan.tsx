import {
  CSSProperties,
  forwardRef,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { useCursorDrag, UseCursorDragStart } from "@/hooks";
import { useStateRef } from "@/hooks/useStateRef";
import { KeyboardKey, useKeyMode } from "@/keys";
import {
  Box,
  Dimensions,
  INFINITE_XY,
  DECIMAL_BOX,
  XY,
  ZERO_BOX,
  ZERO_DIMS,
  ZERO_XY,
} from "@/spatial";
import { BoxScale } from "@/spatial/scale";

import "./ZoomPan.css";

export interface UseZoomPanProps {
  allowZoom?: boolean;
  allowPan?: boolean;
  panHotkey?: KeyboardKey | "";
  zoomHotkey?: KeyboardKey | "";
  onChange?: (box: Box) => void;
  threshold?: Dimensions;
  minZoom?: XY;
  maxZoom?: XY;
  resetOnDoubleClick?: boolean;
}

export interface UseZoomPanReturn {
  maskStyle: CSSProperties;
  containerStyle: CSSProperties;
  mode: Mode | null;
  onDragStart: UseCursorDragStart;
  onDoubleClick: () => void;
  ref: React.RefObject<HTMLDivElement>;
}

type Mode = "zoom" | "pan" | null;

export const useZoomPan = ({
  onChange,
  allowZoom = true,
  allowPan = true,
  panHotkey = "Control",
  zoomHotkey = "",
  threshold = ZERO_DIMS,
  minZoom = ZERO_XY,
  maxZoom = INFINITE_XY,
  resetOnDoubleClick = true,
}: UseZoomPanProps): UseZoomPanReturn => {
  const [maskBox, setMaskBox] = useState<Box>(ZERO_BOX);
  const [stateRef, setStateRef] = useStateRef<Box>(DECIMAL_BOX);
  const canvasRef = useRef<HTMLDivElement>(null);

  const defaultMode = useMemo(() => {
    if (allowZoom && zoomHotkey === "") return "zoom";
    if (allowPan && panHotkey === "") return "pan";
    return null;
  }, [allowZoom, allowPan, zoomHotkey, panHotkey]);

  const mode = useKeyMode<Mode>(
    new Map([
      [zoomHotkey, "zoom"],
      [panHotkey, "pan"],
    ]),
    defaultMode
  );

  const handleDoubleClick = useCallback(() => {
    if (!resetOnDoubleClick) return;
    setStateRef(DECIMAL_BOX);
    onChange?.(DECIMAL_BOX);
  }, [onChange, setStateRef]);

  const handleMove = useCallback(
    (box: Box, key: KeyboardKey): void => {
      if (mode == null || canvasRef.current == null) return;
      const canvas = new Box(canvasRef.current);
      if (mode === "zoom" || key === zoomHotkey) {
        setMaskBox(
          BoxScale.scale(canvas)
            .clamp(canvas)
            .translate({
              x: -canvas.left,
              y: -canvas.top,
            })
            .box(fullSize(threshold, box, canvas))
        );
      } else if (mode === "pan" || key === panHotkey)
        onChange?.(handlePan(box, stateRef.current, canvas));
    },
    [mode, setMaskBox]
  );

  const handleZoom = useCallback(
    (box: Box, prev: Box, canvas: Box) =>
      scale(prev, canvas).box(fullSize(threshold, box, canvas)),
    [threshold]
  );

  const handleEnd = useCallback(
    (box: Box, key: KeyboardKey) => {
      if (canvasRef.current == null) return;
      const canvas = new Box(canvasRef.current);
      setStateRef((prev) => {
        let next: Box | null = null;
        if (mode === "pan" || key === panHotkey) next = handlePan(box, prev, canvas);
        else if (mode === "zoom") next = handleZoom(box, prev, canvas);
        setMaskBox(ZERO_BOX);
        if (next == null) return prev;
        if (next.width < minZoom.x || next.height < minZoom.y) return prev;
        if (next.width > maxZoom.x || next.height > maxZoom.y) return prev;
        onChange?.(next);
        return next;
      });
    },
    [mode, setMaskBox]
  );

  const onDragStart = useCursorDrag({ onMove: handleMove, onEnd: handleEnd });

  const containerStyle: React.CSSProperties = {
    cursor: mode === "zoom" ? "crosshair" : "grab",
  };

  return {
    maskStyle: maskBox.css,
    mode,
    onDragStart,
    ref: canvasRef,
    containerStyle,
    onDoubleClick: handleDoubleClick,
  };
};

type DivProps = React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export interface ZoomPanProps
  extends Omit<UseZoomPanReturn, "ref">,
    Omit<DivProps, "onDragStart" | "onDragEnd" | "onDrag" | "ref" | "onDoubleClick"> {}

export const ZoomPanMask = forwardRef<HTMLDivElement, ZoomPanProps>(
  (
    { maskStyle, onDragStart, mode, style, containerStyle, className, ...props },
    ref
  ): JSX.Element | null => (
    <div
      ref={ref}
      onMouseDown={onDragStart}
      style={{ ...containerStyle, ...style }}
      {...props}
    >
      <div style={maskStyle} className="pluto-zoom-pan-mask" />
    </div>
  )
);
ZoomPanMask.displayName = "ZoomPanMask";

const scale = (prev: Box, canvas: Box): BoxScale =>
  BoxScale.scale(canvas).clamp(canvas).scale(prev);

const handlePan = (box: Box, prev: Box, canvas: Box): Box =>
  BoxScale.translate(scale(prev, canvas).box(box).signedDims).box(prev);

const fullSize = (threshold: Dimensions, box: Box, parent: Box): Box => {
  if (box.height <= threshold.height)
    return new Box(box.left, parent.top, box.width, parent.height);
  if (box.width <= threshold.width)
    return new Box(parent.left, box.top, parent.width, box.height);
  return box;
};
