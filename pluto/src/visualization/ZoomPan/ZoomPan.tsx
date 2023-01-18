import {
  CSSProperties,
  forwardRef,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { useCursorDrag, UseCursorDragStart } from "@/hooks";
import { KeyboardKey, useKeyMode } from "@/keys";
import {
  Box,
  Dimensions,
  Direction,
  INFINITE_XY,
  ONE_XY,
  XY,
  ZERO_BOX,
  ZERO_XY,
} from "@/spatial";

export interface UseZoomPanProps {
  allowZoom?: boolean;
  allowPan?: boolean;
  panHotkey?: KeyboardKey | "";
  zoomHotkey?: KeyboardKey | "";
  onChange?: (box: Box) => void;
  threshold?: XY;
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
  threshold = ZERO_XY,
  minZoom = ZERO_XY,
  maxZoom = INFINITE_XY,
  resetOnDoubleClick = true,
}: UseZoomPanProps): UseZoomPanReturn => {
  const [maskBox, setMaskBox] = useState<Box>(ZERO_BOX);
  const state = useRef<Box>(new Box(ZERO_XY, ONE_XY));
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
    if (canvasRef.current == null || !resetOnDoubleClick) return;
    state.current = new Box(ZERO_XY, ONE_XY);
    onChange?.(state.current);
  }, [onChange]);

  const handleMove = useCallback(
    (box: Box, key: KeyboardKey): void => {
      if (mode == null || canvasRef.current == null) return;
      const canvas = new Box(canvasRef.current.getBoundingClientRect());
      const clamped = box.clampBy(canvas);
      if ((mode === "zoom" && key !== panHotkey) || key === zoomHotkey)
        setMaskBox(clamped);
      else if (mode === "pan" || key === panHotkey) {
        const scaledBox = clamped.toDecimal(canvas).scaleByDims(state.current);
        const next = state.current.translateBySignedDims({
          signedWidth: -scaledBox.signedWidth,
          signedHeight: -scaledBox.signedHeight,
        });
        onChange?.(next);
      }
    },
    [mode, setMaskBox]
  );

  const handleEnd = useCallback(
    (box: Box, key: KeyboardKey) => {
      if (canvasRef.current == null) return;
      const canvas = new Box(canvasRef.current.getBoundingClientRect());
      const decimal = box.clampBy(canvas).toDecimal(canvas);
      if (mode === "pan" || key === panHotkey) {
        const scaledBox = decimal.scaleByDims(state.current);
        state.current = state.current.translateBySignedDims({
          signedWidth: -scaledBox.signedWidth,
          signedHeight: -scaledBox.signedHeight,
        });
        onChange?.(state.current);
        return;
      }
      const fullSize = fullSizeDirection(threshold, box);
      let correctedBox: Box = decimal;
      if (fullSize === "x") correctedBox = new Box(0, decimal.top, 1, decimal.height);
      else if (fullSize === "y")
        correctedBox = new Box(decimal.left, 0, decimal.width, 1);
      const nextBox = correctedBox.scaleByDims(state.current).translate(state.current);
      setMaskBox(ZERO_BOX);
      if (nextBox.width < minZoom.x || nextBox.height < minZoom.y) return;
      if (nextBox.width > maxZoom.x || nextBox.height > maxZoom.y) return;
      state.current = nextBox;
      onChange?.(nextBox);
    },
    [mode, setMaskBox]
  );

  const onDragStart = useCursorDrag({ onMove: handleMove, onEnd: handleEnd });

  const maskStyle: React.CSSProperties = {
    position: "relative",
    width: maskBox.width,
    height: maskBox.height,
    backgroundColor: "rgba(0, 0, 0, 0.5)",
  };

  if (canvasRef.current != null) {
    const canvas = new Box(canvasRef.current.getBoundingClientRect());
    maskStyle.top = maskBox.top - canvas.top;
    maskStyle.left = maskBox.left - canvas.left;
    const fullSize = fullSizeDirection(threshold, maskBox);
    if (fullSize === "y") {
      maskStyle.height = "100%";
      maskStyle.top = 0;
    } else if (fullSize === "x") {
      maskStyle.width = "100%";
      maskStyle.left = 0;
    }
  }

  const containerStyle: React.CSSProperties = {
    cursor: mode === "zoom" ? "crosshair" : "grab",
  };

  return {
    maskStyle,
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
    { maskStyle, onDragStart, mode, style, containerStyle, ...props },
    ref
  ): JSX.Element | null => (
    <div
      ref={ref}
      onMouseDown={onDragStart}
      style={{ ...containerStyle, ...style }}
      {...props}
    >
      <div style={maskStyle} />
    </div>
  )
);
ZoomPanMask.displayName = "ZoomPanMask";

const fullSizeDirection = (threshold: XY, dims: Dimensions): Direction | null => {
  if (dims.height <= threshold.y) return "y";
  if (dims.width <= threshold.x) return "x";
  return null;
};
