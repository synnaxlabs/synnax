// The Gradient component is a typescript react component that renders an input element
// that allows the user to select a color gradient with multiple stops.  The gradient
// is rendered as a very wide and thin (5rem) div element with a background color that
// that follows the multi-stop gradient.  The gradient is rendered as a linear gradient

import "@/color/Gradient.css";

import { box, id } from "@synnaxlabs/x";
import { type ReactElement, useRef, useState } from "react";

import { Align } from "@/align";
import { type color } from "@/color/core";
import { cssString } from "@/color/core/color";
import { Swatch } from "@/color/Swatch";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Input } from "@/input";

interface Stop {
  key: string;
  color: color.Crude;
  position: number;
}

interface GradientProps extends Input.Control<Stop[]> {}

export const G = () => {
  const [stops, setStops] = useState<Stop[]>([]);
  return <Gradient value={stops} onChange={setStops} />;
};

const PICKER_CLS = CSS.B("gradient-picker");
const PREVIEW_CLS = CSS.BE("gradient-picker", "preview");

const stopPosition = (e: React.MouseEvent | MouseEvent): number | null => {
  let t = e.target as HTMLElement;
  if (!t.className.includes(PICKER_CLS) && !t.className.includes(PREVIEW_CLS))
    return null;
  if (t.classList.contains(PREVIEW_CLS)) t = t.closest(`.${PICKER_CLS}`) as HTMLElement;
  const b = box.construct(t);
  const actualX = e.clientX - box.left(b);
  return actualX / box.width(b);
};

export const Gradient = ({ value, onChange }: GradientProps): ReactElement => {
  const sortedStops = value.sort((a, b) => a.position - b.position);
  const grad = buildGradient(sortedStops);
  const prevValue = useSyncedRef(value);
  const handleChange = (stop: Stop) => {
    onChange(prevValue.current.map((s) => (s.key === stop.key ? stop : s)));
  };
  return (
    <div
      className={CSS(PICKER_CLS)}
      style={{
        background: `linear-gradient(to right, ${grad})`,
      }}
      onClick={(e) => {
        const x = stopPosition(e);
        if (x == null) return;
        const newStop: Stop = {
          key: id.id(),
          color: "#000000",
          position: x,
        };
        onChange([...sortedStops, newStop]);
      }}
    >
      {sortedStops.map((stop) => (
        <StopSwatch stop={stop} onChange={handleChange} key={stop.key} />
      ))}
    </div>
  );
};

const buildGradient = (stops: Stop[]): string => {
  if (stops.length === 0) return "white, black";
  return stops
    .map(({ color, position }) => `${cssString(color)} ${position * 100}%`)
    .join(", ");
};

interface StopSwatchProps {
  stop: Stop;
  onChange: (stop: Stop) => void;
}

const StopSwatch = ({ stop, onChange }: StopSwatchProps) => {
  const positionRef = useRef(stop.position);
  const stopElRef = useRef<HTMLDivElement>(null);
  const onDragStart = useCursorDrag({
    onStart: () => {
      positionRef.current = stop.position;
    },
    onMove: (b) => {
      if (stopElRef.current == null) return;
      const picker = stopElRef.current.closest(
        `.${CSS.B("gradient-picker")}`,
      ) as HTMLElement;
      onChange({
        ...stop,
        position: positionRef.current + box.signedWidth(b) / box.width(picker),
      });
    },
  });
  return (
    <Align.Space
      ref={stopElRef}
      className={CSS.BE("gradient-picker", "stop")}
      direction="y"
      style={{
        left: `${stop.position * 100}%`,
      }}
      empty
      align="center"
      justify="center"
      onClick={(e) => {
        e.stopPropagation();
      }}
    >
      <div
        className={CSS.BE("gradient-picker", "drag-region")}
        draggable
        onDragStart={onDragStart}
      >
        <div className={CSS.BE("gradient-picker", "stop-line")} />
      </div>
      <Swatch
        size="small"
        draggable
        key={stop.key}
        value={stop.color}
        onChange={(v: color.Color) => {
          onChange({ ...stop, color: v });
        }}
      />
    </Align.Space>
  );
};
