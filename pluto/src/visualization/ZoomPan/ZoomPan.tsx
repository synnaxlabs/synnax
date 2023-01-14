import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useDrag, UseDragReturn, useKeysHeld } from "@/hooks";
import { KeyboardKey } from "@/keys";
import { Box, CSSBox, PointBox, XY, ZERO_BOX, ZERO_XY } from "@/spatial";

export interface UseZoomPanProps {
  allowZoom?: boolean;
  allowPan?: boolean;
  panHotkey?: KeyboardKey | null;
  zoomHotkey?: KeyboardKey | null;
  onChange?: (zoom: Box, pan: XY) => void;
  threshold?: XY;
}

export interface UseZoomPanReturn extends UseDragReturn {
  zoom: Box;
  threshold: XY;
  mode: Mode | null;
}

type ModeCheck = [Mode, KeyboardKey | null, boolean];

type Mode = "zoom" | "pan";

interface UseZoomPanState {
  pan: XY;
  zoom: Box;
  root: XY | null;
}

export const useZoomPan = ({
  allowZoom = true,
  allowPan = true,
  panHotkey = "Control",
  zoomHotkey = null,
  threshold = ZERO_XY,
  onChange,
}: UseZoomPanProps): UseZoomPanReturn => {
  const [state, setState] = useState<UseZoomPanState>({
    pan: ZERO_XY,
    zoom: ZERO_BOX,
    root: null,
  });

  const { keys } = useKeysHeld(
    useMemo(
      () => [panHotkey, zoomHotkey].filter((key) => key != null) as KeyboardKey[],
      [panHotkey, zoomHotkey]
    )
  );

  const mode: Mode | null = useMemo(
    () =>
      (
        [
          ["zoom", zoomHotkey, allowZoom],
          ["pan", panHotkey, allowPan],
        ] as ModeCheck[]
      )
        .sort(([, a], [, b]) => {
          if (a == null && b != null) return 1;
          if (a != null && b == null) return -1;
          return 0;
        })
        .find(([, key, allowed], i) => {
          if (key == null) return allowed;
          return keys.includes(key) && allowed;
        })?.[0] ?? null,
    [keys]
  );

  useEffect(() => {
    onChange?.(state.zoom, state.pan);
  }, [state]);

  const handleMove = useCallback(
    (e: MouseEvent): void => {
      if (mode == null) return;
      if (mode === "zoom")
        setState((prev) => {
          const point = { x: e.clientX, y: e.clientY };
          return {
            ...prev,
            zoom: new PointBox(prev.root ?? point, point),
            root: prev.root ?? point,
          };
        });
      else if (mode === "pan") {
        setState((prev) => ({
          ...prev,
          pan: {
            x: prev.pan.x + e.movementX,
            y: prev.pan.y + e.movementY,
          },
        }));
      }
    },
    [mode, setState]
  );

  const handleEnd = useCallback(
    () => setState((prev) => ({ ...prev, zoom: ZERO_BOX, root: null })),
    [setState]
  );

  const dragProps = useDrag({ onMove: handleMove, onEnd: handleEnd });

  return { zoom: state.zoom, threshold, mode, ...dragProps };
};

type DivProps = React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export interface ZoomPanProps
  extends UseZoomPanReturn,
    Omit<DivProps, "onDragStart" | "onDragEnd" | "onDrag"> {}

export const ZoomPanMask = ({
  zoom,
  threshold,
  onDragStart,
  mode,
  style,
  ...props
}: ZoomPanProps): JSX.Element | null => {
  const ref = useRef<HTMLDivElement>(null);
  const zoomStyle: React.CSSProperties = {
    position: "relative",
    width: zoom.width,
    height: zoom.height,
    backgroundColor: "rgba(0, 0, 0, 0.5)",
  };
  const zoomContainerStyle: React.CSSProperties = {
    cursor: mode === "zoom" ? "crosshair" : "grab",
    ...style,
  };
  if (ref.current != null) {
    const dBox = new CSSBox(ref.current.getBoundingClientRect());
    zoomStyle.top = zoom.top - dBox.top;
    zoomStyle.left = zoom.left - dBox.left;
    const widthThreshold = zoom.width <= threshold.y;
    const heightThreshold = zoom.height <= threshold.x;
    if (heightThreshold) {
      zoomStyle.height = "100%";
      zoomStyle.top = 0;
    } else if (widthThreshold) {
      zoomStyle.width = "100%";
      zoomStyle.left = 0;
    }
  }
  return (
    <div ref={ref} onMouseDown={onDragStart} style={zoomContainerStyle} {...props}>
      <div style={zoomStyle} />
    </div>
  );
};
