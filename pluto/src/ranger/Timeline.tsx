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
import { type ReactElement, useEffect, useState } from "react";

import { type Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Menu } from "@/menu";
import { moveEnd, moveStart, UNSET } from "@/ranger/move";
import {
  getStage,
  moveToStage,
  type Stage,
  STAGE_ICONS,
  STAGE_NAMES,
} from "@/ranger/stage";
import { describeChanges, TimelineEffect } from "@/ranger/TimelineEffect";
import { Text } from "@/text";

/**
 * The finest unit the row's labels show, chosen by the span between the range's
 * ends: a day-long range reads to the minute, an hour-long one to the second, a
 * second-long one to the millisecond, anything shorter to the microsecond.
 */
const resolutionFor = (span: TimeSpan): TimeSpan => {
  if (span.greaterThanOrEqual(TimeSpan.DAY)) return TimeSpan.MINUTE;
  if (span.greaterThanOrEqual(TimeSpan.MINUTE)) return TimeSpan.SECOND;
  if (span.greaterThanOrEqual(TimeSpan.SECOND)) return TimeSpan.MILLISECOND;
  return TimeSpan.MICROSECOND;
};

/**
 * The resolution for the labels of `range`. An open range is measured to now, and
 * never reads finer than the second its elapsed clock ticks by.
 */
const rangeResolution = ({ start, end }: NumericTimeRange): TimeSpan => {
  const open = end >= UNSET;
  const far = open ? TimeStamp.now().nanoseconds : end;
  const near = start < UNSET ? start : far;
  const resolution = resolutionFor(
    new TimeSpan(BigInt(Math.abs(Math.trunc(far - near)))),
  );
  return open && resolution.lessThan(TimeSpan.SECOND) ? TimeSpan.SECOND : resolution;
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

/**
 * The transitions out of the range's stage, over the effect of the hovered one. It
 * mounts with the menu, so the hover resets each time the menu opens.
 */
const StageMenu = ({
  value,
  onChange,
}: Input.Control<NumericTimeRange>): ReactElement => {
  const [hovered, setHovered] = useState<Stage | null>(null);
  return (
    <>
      <Flex.Box y gap="tiny" className={CSS.BE("stage-button", "items")}>
        <Menu.Menu onChange={(to) => onChange(moveToStage(value, to as Stage))}>
          {TRANSITIONS[getStage(value)].map(({ to, name }) => {
            const ToIcon = STAGE_ICONS[to];
            return (
              <Menu.Item
                key={to}
                itemKey={to}
                onMouseEnter={() => setHovered(to)}
                onMouseLeave={() => setHovered(null)}
                onFocus={() => setHovered(to)}
                onBlur={() => setHovered(null)}
              >
                <ToIcon />
                {name}
              </Menu.Item>
            );
          })}
        </Menu.Menu>
      </Flex.Box>
      <Input.Effect className={CSS.BE("stage-button", "effect")}>
        {hovered == null ? null : (
          <TimelineEffect
            changes={describeChanges(value, moveToStage(value, hovered))}
            resolution={rangeResolution(value)}
          />
        )}
      </Input.Effect>
    </>
  );
};

export interface StageButtonProps extends Input.Control<NumericTimeRange> {
  /** Shows the icon alone, for list rows. */
  iconOnly?: boolean;
  variant?: Button.Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  preview?: boolean;
  className?: string;
}

/**
 * The range's stage as a chip whose menu holds the transitions out of it. A
 * transition writes the timestamps that define the next stage: Start stamps the
 * start, Complete stamps the end, Reopen clears it. The menu says what the hovered
 * transition would change.
 */
export const StageButton = ({
  value,
  onChange,
  iconOnly = false,
  variant = "text",
  level,
  size,
  disabled,
  preview,
  className,
}: StageButtonProps): ReactElement => {
  const [open, setOpen] = useState(false);
  const stage = getStage(value);
  const I = STAGE_ICONS[stage];
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
        preview={preview}
        className={CSS.BE("stage-button", "trigger")}
        tooltip={iconOnly ? STAGE_NAMES[stage] : undefined}
      >
        <I />
        {!iconOnly && STAGE_NAMES[stage]}
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("stage-button", "menu")} background={1}>
        <StageMenu
          value={value}
          onChange={(next) => {
            onChange(next);
            setOpen(false);
          }}
        />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const Dot = ({ level }: { level?: text.Level }): ReactElement => (
  <Text.Text level={level} color={8} className={CSS.BE("range-timeline", "dot")}>
    <Icon.Circle />
  </Text.Text>
);

export interface TimelineProps extends Input.Control<NumericTimeRange> {
  /** The parent range, which enables `T+` expressions and actions. */
  parent?: NumericTimeRange;
  variant?: Input.Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  preview?: boolean;
  className?: string;
}

// setTimeout overflows past this delay and fires at once.
const MAX_TIMEOUT_MS = 2 ** 31 - 1;

/**
 * Re-renders when now crosses the range's next boundary, since the stage and the row's
 * layout are read from the clock.
 */
const useStageBoundary = ({ start, end }: NumericTimeRange): void => {
  const [tick, setTick] = useState(0);
  const now = TimeStamp.now().nanoseconds;
  let boundary: number | null = null;
  if (start < UNSET && start > now) boundary = start;
  else if (end < UNSET && end > now) boundary = end;
  useEffect(() => {
    if (boundary == null) return;
    const remaining = (boundary - TimeStamp.now().nanoseconds) / 1e6;
    const delay = Math.min(Math.max(0, Math.ceil(remaining)) + 1, MAX_TIMEOUT_MS);
    const t = setTimeout(() => setTick((p) => p + 1), delay);
    return () => clearTimeout(t);
  }, [boundary, tick]);
};

/**
 * A range's stage and the timestamps that define it, in one row. To do shows the
 * planned start and, once it is set, the planned end; In progress shows the start, the
 * elapsed time, and the planned end; Completed shows start, end, and duration.
 * Editing a timestamp keeps the other in place; an edit that crosses it slides it to
 * keep the duration. The editor says what the highlighted reading would move and
 * which stage it would land the range in before it is taken. Editing the duration
 * moves the end. The stage chip's menu holds the transitions, which stamp the
 * timestamps.
 */
export const Timeline = ({
  value,
  onChange,
  parent,
  variant = "outlined",
  level,
  size,
  disabled,
  preview,
  className,
}: TimelineProps): ReactElement => {
  useStageBoundary(value);
  const { start, end } = value;
  const stage = getStage(value);
  const scheduled = start < UNSET;
  const ended = end < UNSET;
  const resolution = rangeResolution(value);

  const handleStart = (next: number): void => onChange(moveStart(value, next));
  const handleEnd = (next: number): void => onChange(moveEnd(value, next));
  const handleSpan = (span: number): void => onChange({ start, end: start + span });

  const effectOf = (
    next: NumericTimeRange,
    editing: Input.Bound,
  ): ReactElement | null => {
    const changes = describeChanges(value, next, editing);
    if (changes.length === 0) return null;
    return <TimelineEffect changes={changes} resolution={resolution} />;
  };
  const startEffect = ({ candidate }: { candidate: number }): ReactElement | null =>
    effectOf(moveStart(value, candidate), "start");
  const endEffect = ({ candidate }: { candidate: number }): ReactElement | null =>
    effectOf(moveEnd(value, candidate), "end");

  const startAnchors = { end: ended ? end : undefined, parent };
  const endAnchors = { start, parent };

  const cell = { variant, level, size, disabled, preview, resolution };

  // A cell's content is its value alone, so its name says which end it holds.
  const cellName = (bound: string, instant: number): string =>
    instant >= UNSET
      ? `${bound}, not set`
      : `${bound}, ${Input.formatInstant(instant, TimeStamp.now(), resolution)}`;
  const startName = cellName("Start", start);
  const endName = cellName("End", end);

  const endCell = (
    <>
      {ended && (
        <Text.Text level={level} color={9} className={CSS.BE("range-timeline", "word")}>
          ends
        </Text.Text>
      )}
      <Input.DateTime
        value={end}
        onChange={handleEnd}
        anchors={endAnchors}
        aria-label={endName}
        emptyValue={UNSET}
        placeholder="Set an end time"
        clearLabel="Remove end"
        sharedDay={start}
        bound="end"
        effect={endEffect}
        {...cell}
      />
    </>
  );

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
        preview={preview}
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
            aria-label={startName}
            emptyValue={UNSET}
            placeholder="Set a start time"
            clearLabel="Unschedule"
            bound="start"
            effect={startEffect}
            {...cell}
          />
          {scheduled && endCell}
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
            aria-label={startName}
            bound="start"
            effect={startEffect}
            {...cell}
          />
          {ended && endCell}
          <Dot level={level} />
          <Input.TimeSpan
            value={0}
            onChange={handleSpan}
            elapsedSince={start}
            {...cell}
          />
          {!ended && endCell}
        </>
      )}
      {stage === "completed" && (
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
            aria-label={startName}
            bound="start"
            effect={startEffect}
            {...cell}
          />
          <Text.Text
            level={level}
            color={9}
            className={CSS.BE("range-timeline", "word")}
          >
            ended
          </Text.Text>
          <Input.DateTime
            value={end}
            onChange={handleEnd}
            anchors={endAnchors}
            aria-label={endName}
            sharedDay={start}
            bound="end"
            effect={endEffect}
            {...cell}
          />
          <Dot level={level} />
          <Input.TimeSpan value={end - start} onChange={handleSpan} {...cell} />
        </>
      )}
    </Flex.Box>
  );
};
