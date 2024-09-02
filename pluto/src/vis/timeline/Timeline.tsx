// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/timeline/Timeline.css";

import {
  bounds,
  box,
  CrudeTimeRange,
  CrudeTimeSpan,
  scale,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";
import { c } from "node_modules/vite/dist/node/types.d-aGj9QkWt";
import { ReactElement, useCallback, useRef } from "react";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Ranger } from "@/ranger";
import { Text } from "@/text";

export interface BarSpec {
  key: string;
  label: string;
  timeRange: CrudeTimeRange;
  offset?: CrudeTimeSpan;
  color: Color.Crude;
}

export interface TimelineProps extends Align.SpaceProps, Pick<BarProps, "onTranslate"> {
  bars: BarSpec[];
}

export const Timeline = ({ className, bars, onTranslate, ...props }: TimelineProps) => {
  const maxBounds = bounds.max(
    bars.map((b) => {
      const tr = new TimeRange(b.timeRange);
      return {
        lower: Number(tr.start.valueOf()),
        upper: Number(tr.end.valueOf()),
      };
    }),
  );
  const s = scale.Scale.scale<number>(maxBounds).scale(0, 1);
  return (
    <Align.Space
      direction="y"
      className={CSS(CSS.B("timeline"), className)}
      size={0.5}
      {...props}
    >
      {bars.map((spec) => (
        <Track
          key={spec.key}
          bars={[spec]}
          dataToDecimalScale={s}
          onTranslate={onTranslate}
        />
      ))}
    </Align.Space>
  );
};

export interface TrackProps
  extends Align.SpaceProps,
    Pick<BarProps, "dataToDecimalScale" | "onTranslate"> {
  bars: BarSpec[];
}

export const Track = ({
  className,
  bars,
  dataToDecimalScale,
  onTranslate,
  ...props
}: TrackProps): ReactElement => (
  <Align.Space
    direction="x"
    className={CSS(CSS.B("timeline-track"), className)}
    size={0.5}
    {...props}
  >
    {bars.map((spec) => (
      <Bar
        key={spec.key}
        spec={spec}
        dataToDecimalScale={dataToDecimalScale}
        onTranslate={onTranslate}
      />
    ))}
  </Align.Space>
);

export interface BarProps extends Omit<Align.SpaceProps, "color"> {
  spec: BarSpec;
  dataToDecimalScale: scale.Scale<number>;
  onTranslate?: (bar: string, offset: TimeSpan) => void;
}

export const Bar = ({
  dataToDecimalScale,
  className,
  spec: { key, label, timeRange, color, offset = 0 },
  onTranslate,
  ...props
}: BarProps) => {
  const tr = new TimeRange(timeRange);
  const off = new TimeSpan(offset);
  const left = dataToDecimalScale.pos(Number(tr.start.valueOf() + off.valueOf()));
  const right = dataToDecimalScale.pos(Number(tr.end.valueOf() + off.valueOf()));
  const parsedColor = new Color.Color(color);
  const barRef = useRef<HTMLDivElement>(null);
  const bOffsetRef = useSyncedRef(off);
  const offsetRef = useRef<TimeSpan>(off);
  const startDrag = useCursorDrag({
    onStart: useCallback(() => {
      offsetRef.current = bOffsetRef.current;
    }, []),
    onMove: useCallback((b: box.Box) => {
      const barBox = box.construct(barRef.current);
      const width = box.signedWidth(b);
      const totalSize = box.width(barBox) / (right - left);
      const dragSizeDecimal = width / totalSize;
      onTranslate?.(
        key,
        new TimeSpan(dataToDecimalScale.reverse().dim(dragSizeDecimal)).add(
          offsetRef.current,
        ),
      );
    }, []),
  });
  return (
    <Align.Space
      draggable
      onDragStart={startDrag}
      direction="x"
      ref={barRef}
      className={CSS(className, CSS.B("timeline-bar"))}
      bordered
      rounded
      style={{
        left: `${left * 100}%`,
        right: `${(1 - right) * 100}%`,
        // @ts-expect-error
        "--bar-color": parsedColor.rgbString,
      }}
      {...props}
    >
      <Text.Text level="p">{label}</Text.Text>
      <Ranger.TimeRangeChip level="small" timeRange={tr} />
    </Align.Space>
  );
};
