import { type schematic } from "@synnaxlabs/client";
import { useCursorDrag } from "@synnaxlabs/pluto";
import { box, location, scale, xy } from "@synnaxlabs/x";
import { useRef } from "react";

import { CSS } from "@/css";

export interface HandleProps {
  handle: schematic.symbol.Handle;
  selectedHandle: string | undefined;
  svgBox: box.Box;
  containerBox: box.Box;
  onSelect: (handleKey: string) => void;
  onDrag: (handleKey: string, position: xy.XY) => void;
}

const Handle = ({
  handle,
  selectedHandle,
  svgBox,
  containerBox,
  onSelect,
  onDrag,
}: HandleProps) => {
  const pos = scale.XY.scale(box.reRoot(box.DECIMAL, location.TOP_LEFT))
    .scale(box.construct(xy.ZERO, box.dims(svgBox)))
    .translate(xy.translation(box.topLeft(containerBox), box.topLeft(svgBox)))
    .reBound(box.construct(xy.ZERO, box.dims(containerBox)))
    .scale(box.reRoot(box.DECIMAL, location.TOP_LEFT))
    .magnify({ x: 100, y: 100 })
    .pos(handle.position);
  const isSelected = selectedHandle === handle.key;
  const positionRef = useRef(handle.position);
  const onDragStart = useCursorDrag({
    onStart: () => (positionRef.current = handle.position),
    onMove: (b) => {
      const box1 = box.construct(xy.ZERO, box.dims(svgBox));
      const box2 = box.reRoot(box.DECIMAL, location.TOP_LEFT);
      const nextPos = scale.XY.scale(box1)
        .scale(box2)
        .translate(positionRef.current)
        .clamp(box2)
        .pos(xy.construct(box.signedDims(b)));
      onDrag(handle.key, nextPos);
    },
  });

  return (
    <div
      key={handle.key}
      onDragStart={onDragStart}
      className={CSS(
        CSS.BE("schematic", "handle", "preview"),
        isSelected && CSS.M("selected"),
      )}
      style={{
        position: "absolute",
        left: `${pos.x}%`,
        top: `${pos.y}%  `,
      }}
      draggable
      onMouseDown={(e) => {
        e.stopPropagation();
        onSelect(handle.key);
      }}
    />
  );
};

export const handleScale = (svgElement: SVGSVGElement) => {
  const svgBox = box.construct(svgElement);
  const windowBox = box.construct(document.documentElement);
  return scale.XY.scale(windowBox)
    .clamp(svgBox)
    .translate(xy.scale(box.topLeft(svgBox), -1))
    .reBound(box.construct({ x: 0, y: 0 }, box.dims(svgBox)))
    .scale(box.reRoot(box.DECIMAL, location.TOP_LEFT));
};

export interface HandleOverlayProps {
  handles: schematic.symbol.Handle[];
  selectedHandle: string | undefined;
  svgBox: box.Box;
  containerBox: box.Box;
  onSelect: (handleKey: string) => void;
  onDrag: (handleKey: string, position: xy.XY) => void;
}

export const HandleOverlay = ({
  handles,
  selectedHandle,
  svgBox,
  containerBox,
  onSelect,
  onDrag,
}: HandleOverlayProps) => (
  <div
    style={{
      position: "absolute",
      inset: 0,
      pointerEvents: "none",
      zIndex: 999,
    }}
  >
    {handles.map((handle) => (
      <Handle
        key={handle.key}
        handle={handle}
        selectedHandle={selectedHandle}
        svgBox={svgBox}
        containerBox={containerBox}
        onSelect={onSelect}
        onDrag={onDrag}
      />
    ))}
  </div>
);
