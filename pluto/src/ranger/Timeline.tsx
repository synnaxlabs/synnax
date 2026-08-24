// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/Timeline.css";

import { type NumericTimeRange, type text, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { type Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Menu } from "@/menu";
import {
  getStage,
  type Stage,
  STAGE_ICONS,
  STAGE_NAMES,
  wrapNumericTimeRangeToStage,
} from "@/ranger/stage";
import { Text } from "@/text";

/** An open start or end. */
const UNSET = TimeStamp.MAX.nanoseconds;

/**
 * The finest unit the row's labels show, chosen by the span between the range's
 * ends: a day-long range reads to the minute, an hour-long one to the second, a
 * second-long one to the millisecond, anything shorter to the microsecond.
 */
export const resolutionFor = (span: TimeSpan): TimeSpan => {
  if (span.greaterThanOrEqual(TimeSpan.DAY)) return TimeSpan.MINUTE;
  if (span.greaterThanOrEqual(TimeSpan.MINUTE)) return TimeSpan.SECOND;
  if (span.greaterThanOrEqual(TimeSpan.SECOND)) return TimeSpan.MILLISECOND;
  return TimeSpan.MICROSECOND;
};

/**
 * Commits a start edit. A start moved past the end drags the end with it, keeping
 * the duration. Unscheduling clears the end too.
 */
export const moveStart = (
  { start, end }: NumericTimeRange,
  next: number,
): NumericTimeRange => {
  if (next >= UNSET) return { start: UNSET, end: UNSET };
  if (end < UNSET && next > end) return { start: next, end: next + (end - start) };
  return { start: next, end };
};

/**
 * Commits an end edit. An end moved before the start drags the start with it,
 * keeping the duration; an open range has none to keep, so the start clamps.
 */
export const moveEnd = (
  { start, end }: NumericTimeRange,
  next: number,
): NumericTimeRange => {
  if (next < start)
    return { start: end < UNSET ? next - (end - start) : next, end: next };
  return { start, end: next };
};

interface Transition {
  to: Stage;
  name: string;
}

/** The transitions offered from each stage, most common first. */
const TRANSITIONS: Record<Stage, Transition[]> = {
  to_do: [
    { to: "in_progress", name: "Start" },
    { to: "completed", name: "Complete" },
  ],
  in_progress: [
    { to: "completed", name: "Complete" },
    { to: "to_do", name: "Move to to do" },
  ],
  completed: [{ to: "in_progress", name: "Reopen" }],
};

export interface StageButtonProps extends Input.Control<NumericTimeRange> {
  /** Shows the icon alone, for list rows. */
  iconOnly?: boolean;
  variant?: Button.Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  className?: string;
}

/**
 * The range's stage as a chip whose menu holds the transitions out of it. A
 * transition writes the timestamps that define the next stage: Start stamps the
 * start, Complete stamps the end, Reopen clears it.
 */
export const StageButton = ({
  value,
  onChange,
  iconOnly = false,
  variant = "text",
  level,
  size,
  disabled,
  className,
}: StageButtonProps): ReactElement => {
  const [open, setOpen] = useState(false);
  const stage = getStage(value);
  const { onChange: transition } = wrapNumericTimeRangeToStage({ value, onChange });
  const I = STAGE_ICONS[stage];
  const handleSelect = useCallback(
    (to: string) => {
      transition(to as Stage);
      setOpen(false);
    },
    [transition],
  );
  return (
    <Dialog.Frame
      variant="floating"
      visible={open}
      onVisibleChange={setOpen}
      className={CSS.cls(CSS.B("stage-button"), className)}
    >
      <Dialog.Trigger
        hideCaret
        variant={variant}
        level={level}
        size={size}
        disabled={disabled}
        className={CSS.BE("stage-button", "trigger")}
        tooltip={iconOnly ? STAGE_NAMES[stage] : undefined}
      >
        <I />
        {!iconOnly && STAGE_NAMES[stage]}
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("stage-button", "menu")} background={1}>
        <Flex.Box y gap="tiny" className={CSS.BE("stage-button", "items")}>
          <Menu.Menu onChange={handleSelect}>
            {TRANSITIONS[stage].map(({ to, name }) => {
              const ToIcon = STAGE_ICONS[to];
              return (
                <Menu.Item key={to} itemKey={to}>
                  <ToIcon />
                  {name}
                </Menu.Item>
              );
            })}
          </Menu.Menu>
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const Arrow = ({ level }: { level?: text.Level }): ReactElement => (
  <Text.Text level={level} color={7} className={CSS.BE("range-timeline", "arrow")}>
    <Icon.Arrow.Right />
  </Text.Text>
);

export interface TimelineProps extends Input.Control<NumericTimeRange> {
  /** The parent range, which enables `T+` expressions and actions. */
  parent?: NumericTimeRange;
  variant?: Input.Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  className?: string;
}

/**
 * A range's stage and the timestamps that define it, in one row. To do shows the
 * planned start; In progress shows the start, the elapsed time, and a planned end if
 * one is set; Completed shows start, end, and duration. Editing a timestamp keeps
 * the other in place; an edit that crosses it slides it to keep the duration, and
 * the editor warns with the destination first. Editing the duration moves the end.
 * The stage chip's menu holds the transitions, which stamp the timestamps.
 */
export const Timeline = ({
  value,
  onChange,
  parent,
  variant = "outlined",
  level,
  size,
  disabled,
  className,
}: TimelineProps): ReactElement => {
  const { start, end } = value;
  const stage = getStage(value);
  const scheduled = start < UNSET;

  // Labels read to the unit that separates the range's ends: for a running or
  // planned range, the far end is now.
  const far = end < UNSET ? end : TimeStamp.now().nanoseconds;
  const near = scheduled ? start : far;
  const resolution = resolutionFor(
    new TimeSpan(BigInt(Math.abs(Math.trunc(far - near)))),
  );

  const handleStart = useCallback(
    (next: number) => onChange(moveStart(value, next)),
    [onChange, value],
  );
  const handleEnd = useCallback(
    (next: number) => onChange(moveEnd(value, next)),
    [onChange, value],
  );
  const handleSpan = useCallback(
    (span: number) => onChange({ start, end: start + span }),
    [onChange, start],
  );

  const slideStart = useCallback(
    (next: number) => {
      const moved = moveStart(value, next).end;
      return moved !== end && moved < UNSET ? moved : undefined;
    },
    [value, end],
  );
  const slideEnd = useCallback(
    (next: number) => {
      const moved = moveEnd(value, next).start;
      return moved !== start ? moved : undefined;
    },
    [value, start],
  );

  const startAnchors = useMemo(
    () => ({ end: end < UNSET ? end : undefined, parent }),
    [end, parent],
  );
  const endAnchors = useMemo(() => ({ start, parent }), [start, parent]);

  const cell = { variant, level, size, disabled, resolution };

  return (
    <Flex.Box
      x
      wrap
      align="center"
      gap="small"
      className={CSS.cls(CSS.B("range-timeline"), className)}
    >
      <StageButton
        value={value}
        onChange={onChange}
        variant={variant === "outlined" ? "outlined" : "text"}
        level={level}
        size={size}
        disabled={disabled}
      />
      {stage === "to_do" && (
        <>
          {scheduled && (
            <Text.Text
              level={level}
              color={9}
              className={CSS.BE("range-timeline", "word")}
            >
              starts
            </Text.Text>
          )}
          <Input.DateTime
            value={start}
            onChange={handleStart}
            anchors={startAnchors}
            slide={slideStart}
            emptyValue={UNSET}
            placeholder="Set a start time"
            clearLabel="Unschedule"
            role="start"
            {...cell}
          />
        </>
      )}
      {stage === "in_progress" && (
        <>
          <Text.Text
            level={level}
            color={9}
            className={CSS.BE("range-timeline", "word")}
          >
            started
          </Text.Text>
          <Input.DateTime
            value={start}
            onChange={handleStart}
            anchors={startAnchors}
            slide={slideStart}
            role="start"
            {...cell}
          />
          <Input.TimeSpan
            value={0}
            onChange={handleSpan}
            elapsedSince={start}
            {...cell}
          />
          {end < UNSET && (
            <>
              <Text.Text
                level={level}
                color={9}
                className={CSS.BE("range-timeline", "word")}
              >
                ends
              </Text.Text>
              <Input.DateTime
                value={end}
                onChange={handleEnd}
                anchors={endAnchors}
                slide={slideEnd}
                emptyValue={UNSET}
                clearLabel="Remove end"
                sharedDay={start}
                role="end"
                {...cell}
              />
            </>
          )}
        </>
      )}
      {stage === "completed" && (
        <>
          <Input.DateTime
            value={start}
            onChange={handleStart}
            anchors={startAnchors}
            slide={slideStart}
            role="start"
            {...cell}
          />
          <Arrow level={level} />
          <Input.DateTime
            value={end}
            onChange={handleEnd}
            anchors={endAnchors}
            slide={slideEnd}
            sharedDay={start}
            role="end"
            {...cell}
          />
          <Input.TimeSpan value={end - start} onChange={handleSpan} {...cell} />
        </>
      )}
    </Flex.Box>
  );
};
