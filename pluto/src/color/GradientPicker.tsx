// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/GradientPicker.css";

import { box, clamp, color, id, scale } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { Swatch } from "@/color/Swatch";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type Input } from "@/input";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { stopPropagation } from "@/util/event";

interface GradientProps extends Input.Control<color.Stop[]> {
  scale?: scale.Scale<number>;
}

const SWITCH_THRESHOLD = 0.05;

const switchStops = (stops: color.Stop[]): color.Stop[] =>
  stops.map((stop, i) => {
    if (i === 0) {
      if (stop.switched === true) stop.switched = false;
      return stop;
    }
    const prev = stops[i - 1];
    const delta = Math.abs(stop.position - prev.position);
    if (delta < SWITCH_THRESHOLD && prev.switched !== true) stop.switched = true;
    if (delta > SWITCH_THRESHOLD && stop.switched === true) stop.switched = false;
    return stop;
  });

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
  const sortedStops = switchStops(value.sort((a, b) => a.position - b.position));
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
        style={{ background: `linear-gradient(to right, ${grad})` }}
        onClick={(e) => {
          const x = stopPosition(e);
          if (x == null) return;
          const newStop: color.Stop = {
            key: id.create(),
            color: "#FFFFFF",
            position: x,
          };
          onChange([...sortedStops, newStop]);
        }}
      >
        {sortedStops.map((stop, i) => (
          <StopSwatch
            stop={stop}
            nextStop={sortedStops[i + 1] ?? null}
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
  if (stops.length === 0) return "";
  return stops
    .map(({ color: c, position }, i) => {
      if (i === 0)
        return `#00000000 ${position * 100}%, ${color.cssString(c)} ${position * 100}%`;
      const prevColor = stops[i - 1].color;
      return `${color.cssString(prevColor)} ${position * 100}%, ${color.cssString(c)} ${
        position * 100
      }%`;
    })
    .join(", ");
};

interface StopSwatchProps {
  stop: color.Stop;
  nextStop: color.Stop | null;
  onChange: (stop: color.Stop) => void;
  onDelete: (key: string) => void;
  scale: scale.Scale<number>;
}

const StopSwatch = ({ stop, onChange, nextStop, onDelete, scale }: StopSwatchProps) => {
  const positionRef = useRef(stop.position);
  const { switched } = stop;
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
  const [visible, setVisible, visibleRef] = useCombinedStateAndRef<boolean>(false);
  Triggers.use({
    triggers: [["Delete"]],
    callback: ({ stage }) => {
      if (!visibleRef.current || stage !== "end") return;
      onDelete(stop.key);
    },
  });

  return (
    <Flex.Box
      ref={stopElRef}
      className={CSS(CSS.BE("gradient-picker", "stop"), switched && CSS.M("switched"))}
      y
      style={{
        left: `${stop.position * 100}%`,
        width: `${(nextStop?.position ?? 1) * 100 - stop.position * 100}%`,
      }}
      empty
      onClick={stopPropagation}
    >
      <Flex.Box
        y
        className={CSS.BE("gradient-picker", "drag-region")}
        draggable
        onDragStart={onDragStart}
        empty
      >
        <div
          className={CSS.BE("gradient-picker", "stop-line")}
          onDoubleClick={(e) => {
            e.stopPropagation();
            onDelete(stop.key);
          }}
        />
        <Text.Editable
          level="small"
          value={scale.pos(stop.position).toFixed(2)}
          onChange={(v) => {
            onChange({ ...stop, position: scale.reverse().pos(Number(v)) });
          }}
        />
      </Flex.Box>
      <Swatch
        size="small"
        draggable
        key={stop.key}
        value={stop.color}
        onVisibleChange={setVisible}
        visible={visible}
        onChange={(v: color.Color) => onChange({ ...stop, color: v })}
      />
    </Flex.Box>
  );
};
