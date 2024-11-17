// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/GradientPicker.css";

import { box, clamp, id, scale } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { Align } from "@/align";
import { type color } from "@/color/core";
import { cssString } from "@/color/core/color";
import { Swatch } from "@/color/Swatch";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Input } from "@/input";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

interface GradientProps extends Input.Control<color.Stop[]> {
  scale?: scale.Scale<number>;
}

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

export const GradientPicker = ({
  value,
  onChange,
  scale: scl = scale.Scale.IDENTITY,
}: GradientProps): ReactElement => {
  const sortedStops = value.sort((a, b) => a.position - b.position);
  const grad = buildGradient(sortedStops);
  const prevValue = useSyncedRef(value);
  const handleChange = (stop: color.Stop) => {
    onChange(prevValue.current.map((s) => (s.key === stop.key ? stop : s)));
  };
  const handleDelete = (key: string) => {
    onChange(prevValue.current.filter((s) => s.key !== key));
  };
  return (
    <div className={CSS(PICKER_CLS)}>
      <div
        className={CSS(CSS.BE("gradient-picker", "bar"))}
        style={{
          background: `linear-gradient(to right, ${grad})`,
        }}
        onClick={(e) => {
          const x = stopPosition(e);
          if (x == null) return;
          const newStop: color.Stop = {
            key: id.id(),
            color: "#FFFFFF",
            position: x,
          };
          onChange([...sortedStops, newStop]);
        }}
      >
        {sortedStops.map((stop) => (
          <StopSwatch
            stop={stop}
            onChange={handleChange}
            onDelete={handleDelete}
            key={stop.key}
            scale={scl}
          />
        ))}
      </div>
    </div>
  );
};

const buildGradient = (stops: color.Stop[]): string => {
  if (stops.length === 0) return "white, black";
  return stops
    .map(({ color, position }) => `${cssString(color)} ${position * 100}%`)
    .join(", ");
};

interface StopSwatchProps {
  stop: color.Stop;
  onChange: (stop: color.Stop) => void;
  onDelete: (key: string) => void;
  scale: scale.Scale<number>;
}

const StopSwatch = ({ stop, onChange, onDelete, scale }: StopSwatchProps) => {
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
        position: clamp(
          positionRef.current + box.signedWidth(b) / box.width(picker),
          0,
          1,
        ),
      });
    },
  });
  const visibleRef = useRef(false);
  Triggers.use({
    triggers: [["Delete"]],
    callback: ({ stage }) => {
      if (!visibleRef.current || stage !== "end") return;
      onDelete(stop.key);
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
      onClick={(e) => e.stopPropagation()}
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
        onVisibleChange={(v) => {
          visibleRef.current = v;
        }}
        onChange={(v: color.Color) => {
          onChange({ ...stop, color: v });
        }}
      />
      <Text.Editable
        level="small"
        value={scale.pos(stop.position).toFixed(2)}
        onChange={(v) => {
          onChange({ ...stop, position: scale.reverse().pos(Number(v)) });
        }}
      />
    </Align.Space>
  );
};
