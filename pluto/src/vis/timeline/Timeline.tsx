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
  CrudeTimeStamp,
  scale,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { re } from "mathjs";
import {
  createContext,
  PropsWithChildren,
  ReactElement,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { useRequiredContext } from "@/hooks/useRequiredContext";
import { Ranger } from "@/ranger";
import { Text } from "@/text";

export interface BarSpec {
  key: string;
  label: string;
  timeRange: CrudeTimeRange;
  offset?: CrudeTimeSpan;
  color: Color.Crude;
}

export interface TimelineProps extends Align.SpaceProps {}

export interface ContextValue {
  dataToDecimalScale: scale.Scale<number>;
  register: (key: string, timeRange: CrudeTimeRange) => void;
}

export const Context = createContext<ContextValue | null>(null);

export const useContext = () => useRequiredContext(Context);

export interface ProviderProps extends PropsWithChildren<{}> {}

export const Provider = ({ children }: ProviderProps) => {
  const [ds, setDS] = useState<scale.Scale<number>>(scale.Scale.scale<number>(0, 1));
  const barsRef = useRef<Record<string, CrudeTimeRange>>({});
  const updateScale = useCallback(() => {
    const bars = Object.values(barsRef.current);
    const maxBounds = bounds.max(
      bars.map((b) => {
        const tr = new TimeRange(b);
        return {
          lower: Number(tr.start.valueOf()),
          upper: Number(tr.end.valueOf()),
        };
      }),
    );
    setDS(scale.Scale.scale<number>(maxBounds).scale(0, 1));
  }, []);

  const register = useCallback(
    (key: string, timeRange: CrudeTimeRange) => {
      barsRef.current[key] = timeRange;
      updateScale();
    },
    [updateScale],
  );

  const ctxValue = useMemo(() => ({ dataToDecimalScale: ds, register }), [ds, re]);

  return <Context.Provider value={ctxValue}>{children}</Context.Provider>;
};

export const Timeline = ({ className, ...props }: TimelineProps) => {
  return (
    <Provider>
      <Align.Space
        direction="y"
        className={CSS(CSS.B("timeline"), className)}
        size="small"
        {...props}
      />
    </Provider>
  );
};

export interface TrackProps extends Align.SpaceProps, Pick<BarProps, "onTranslate"> {}

export const Track = ({
  className,
  onTranslate,
  ...props
}: TrackProps): ReactElement => (
  <Align.Space
    direction="x"
    className={CSS(CSS.B("timeline-track"), className)}
    size={0.5}
    {...props}
  />
);

export interface BarProps
  extends Omit<Align.SpaceProps, "color">,
    Omit<BarSpec, "key"> {
  onTranslate?: (bar: string, offset: TimeSpan) => void;
}

export const Bar = ({
  className,
  key,
  label,
  timeRange,
  color,
  offset = 0,
  onTranslate,
  ...props
}: BarProps) => {
  const { dataToDecimalScale, register } = useContext();
  const tr = new TimeRange(timeRange);
  const off = new TimeSpan(offset);
  const left = dataToDecimalScale.pos(Number(tr.start.valueOf() + off.valueOf()));
  const right = dataToDecimalScale.pos(Number(tr.end.valueOf() + off.valueOf()));

  const parsedColor = new Color.Color(color);
  const barRef = useRef<HTMLDivElement>(null);

  const k = useId();

  useEffect(() => {
    register(k, timeRange);
  }, [k, register, timeRange]);

  const bOffsetRef = useSyncedRef(off);
  const offsetRef = useRef<TimeSpan>(off);
  const startDrag = useCursorDrag({
    onStart: useCallback(() => {
      offsetRef.current = bOffsetRef.current;
    }, []),
    onMove: useCallback((b: box.Box, _: any, ev: MouseEvent) => {
      const barBox = box.construct(barRef.current);
      let width = box.signedWidth(b);
      const totalSize = box.width(barBox) / (right - left);
      if (ev.altKey) width *= 0.1;
      const dragSizeDecimal = width / totalSize;
      onTranslate?.(
        "",
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

export interface CursorProps {
  position: CrudeTimeStamp;
  onPositionChange?: (position: CrudeTimeStamp) => void;
}

export const Cursor = ({ position, onPositionChange }: CursorProps) => {
  const { dataToDecimalScale } = useContext();
  const pos = new TimeStamp(position);
  const left = dataToDecimalScale.pos(Number(pos.valueOf()));
  const [hoverPos, setHoverPos] = useState<number | null>(null);

  return (
    <div
      className={CSS.BE("cursor", "container")}
      onMouseMove={(e) => {
        const rect = e.currentTarget.getBoundingClientRect();
        setHoverPos((e.clientX - rect.left) / rect.width);
      }}
      onMouseLeave={() => setHoverPos(null)}
      onClick={(e) => {
        const rect = e.currentTarget.getBoundingClientRect();
        const pos = (e.clientX - rect.left) / rect.width;
        const ts = dataToDecimalScale.reverse().pos(pos);
        onPositionChange?.(ts);
      }}
    >
      {hoverPos != null && (
        <div
          className={CSS(CSS.B("cursor"), CSS.B("cursor-preview"))}
          style={{
            left: `${hoverPos * 100}%`,
          }}
        />
      )}
      <div
        className={CSS.B("cursor")}
        style={{
          left: `${left * 100}%`,
        }}
      />
    </div>
  );
};
