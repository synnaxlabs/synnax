// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import {
  type NumericTimeRange,
  type text,
  type TimeSpan as XTimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import {
  type ReactElement,
  type ReactNode,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/input/Text";
import { Cell } from "@/input/time/Cell";
import {
  type Anchors,
  describeDay,
  formatTime,
  formatTimeStamp,
  fromNumeric,
  nudge,
  parseTimeStamp,
  sameDay,
  unitAt,
  zoneName,
} from "@/input/time/grammar";
import { type Role, type Suggestion, suggestTimeStamps } from "@/input/time/suggest";
import { type Control, type Variant } from "@/input/types";
import { Menu } from "@/menu";
import { Text as BaseText } from "@/text";

/** The instants a cell may anchor typed expressions on, as form values. */
export interface DateTimeAnchors {
  /** The other end of the range the cell belongs to. */
  start?: number;
  end?: number;
  /** The parent range, the `T` in `T+3.2s`. */
  parent?: NumericTimeRange;
}

export interface DateTimeProps extends Control<number> {
  anchors?: DateTimeAnchors;
  /**
   * An instant whose day is already shown beside this cell; the label drops its own
   * day when the two match.
   */
  sharedDay?: number;
  /** The cell's place in a range; decides what a bare duration or time means. */
  role?: Role;
  /** The finest unit the label shows; the tooltip and editor keep every digit. */
  resolution?: XTimeSpan;
  variant?: Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  preview?: boolean;
  className?: string;
  /**
   * Renders a value equal to this as an empty cell and commits it when the cell is
   * cleared.
   */
  emptyValue?: number;
  /** What the empty cell says at rest, as a prompt: `Set a start time`. */
  placeholder?: string;
  /** The action that clears the cell back to `emptyValue`. */
  clearLabel?: string;
}

const describeOffset = (value: TimeStamp, from: TimeStamp): string => {
  const span = value.span(from);
  if (span.isZero) return "";
  const text = span.toString("semantic");
  return value.after(from) ? `${text} from now` : `${text} ago`;
};

const describeParent = (value: TimeStamp, parent: TimeStamp): string => {
  const sign = value.before(parent) ? "-" : "+";
  return `T${sign}${value.span(parent).toString("full") || "0s"}`;
};

const FIXED_LAYOUT_RE =
  /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{3}(?: \d{3}){0,2})?$/;

interface SuggestionsProps {
  text: string;
  suggestions: Suggestion[];
  selected: string | undefined;
  anchors: Anchors;
  onSelect: (key: string) => void;
}

const Suggestions = ({
  text,
  suggestions,
  selected,
  anchors,
  onSelect,
}: SuggestionsProps): ReactElement => {
  if (text.trim().length === 0)
    return (
      <BaseText.Text level="small" color={9} className={CSS.BE("datetime", "note")}>
        Try 14:05, tomorrow 3pm, now - 5m, or 2h
      </BaseText.Text>
    );
  if (suggestions.length === 0)
    return (
      <BaseText.Text
        level="small"
        status="error"
        className={CSS.BE("datetime", "note")}
      >
        Not a time
      </BaseText.Text>
    );
  return (
    <Menu.Menu value={selected} onChange={onSelect}>
      {suggestions.map(({ key, value, reading }) => {
        // A reading relative to now already says what the offset hint would.
        const relative = /\b(now|ago)$/.test(reading);
        const hints = relative ? [] : [describeOffset(value, anchors.now)];
        if (anchors.parent != null) hints.push(describeParent(value, anchors.parent));
        return (
          <Menu.Item
            key={key}
            itemKey={key}
            className={CSS.BE("datetime", "suggestion")}
          >
            <Flex.Box y gap="tiny" grow>
              <Flex.Box x align="center" justify="between" gap="medium">
                <span>
                  <span className={CSS.BE("datetime", "day")}>
                    {describeDay(value, anchors.now)}
                  </span>{" "}
                  {formatTime(value)}
                </span>
                <BaseText.Text level="small" color={9}>
                  {hints.filter((h) => h.length > 0).join(" · ")}
                </BaseText.Text>
              </Flex.Box>
              <BaseText.Text level="small" color={9} overflow="ellipsis">
                {reading}
              </BaseText.Text>
            </Flex.Box>
          </Menu.Item>
        );
      })}
    </Menu.Menu>
  );
};

interface ActionProps {
  itemKey: string;
  icon: ReactElement;
  label: string;
  hint: string;
}

const Action = ({ itemKey, icon, label, hint }: ActionProps): ReactElement => (
  <Menu.Item itemKey={itemKey} className={CSS.BE("datetime", "action")}>
    {icon}
    {label}
    <BaseText.Text color={9} className={CSS.BE("datetime", "hint")}>
      {hint}
    </BaseText.Text>
  </Menu.Item>
);

/**
 * A date-time cell. The value is nanoseconds since the Unix epoch. At rest it reads
 * as content (`Today 14:05:32`); clicking opens an editor with the instant in a fixed
 * local layout, where typing replaces it with an expression and Up and Down nudge
 * the unit under the caret (Shift for ten). Enter or a click outside commits.
 */
export const DateTime = ({
  value,
  onChange,
  anchors: propsAnchors,
  emptyValue,
  placeholder,
  sharedDay,
  role,
  resolution,
  clearLabel = "Clear",
  variant = "outlined",
  className,
  disabled,
  preview,
  level,
  size,
}: DateTimeProps): ReactElement => {
  const [open, setOpen] = useState(false);
  const [text, setTextState] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const revertRef = useRef(false);
  const caretRef = useRef<number | null>(null);

  const isEmpty = emptyValue != null && value >= emptyValue;

  const anchors: Anchors = { now: TimeStamp.now() };
  if (propsAnchors?.start != null) anchors.start = new TimeStamp(propsAnchors.start);
  if (propsAnchors?.end != null) anchors.end = new TimeStamp(propsAnchors.end);
  if (propsAnchors?.parent != null)
    anchors.parent = new TimeStamp(propsAnchors.parent.start);

  const stamp = isEmpty ? null : fromNumeric(value);
  const formatted = stamp == null ? "" : formatTimeStamp(stamp);

  const setText = useCallback((next: string) => {
    setTextState(next);
    setSelected(0);
  }, []);

  // Readings are re-derived on each edit, so `now` inside them is that moment.
  const suggestions = useMemo(
    () => suggestTimeStamps(text, { anchors, current: stamp ?? undefined, role }),
    [text, propsAnchors?.start, propsAnchors?.end, propsAnchors?.parent, value, role],
  );
  const chosen = suggestions[Math.min(selected, suggestions.length - 1)];

  const commit = useCallback(
    (next: number) => {
      const rounded = Number(fromNumeric(next).valueOf());
      if (rounded !== value) onChange(rounded);
    },
    [onChange, value],
  );

  const finish = useCallback(() => {
    const cleared = text.trim().length === 0;
    if (revertRef.current) revertRef.current = false;
    else if (cleared && emptyValue != null) commit(emptyValue);
    else if (!cleared && chosen != null) commit(Number(chosen.value.valueOf()));
    setOpen(false);
  }, [text, emptyValue, chosen, commit]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (next) {
        setText(formatted);
        setOpen(true);
      } else finish();
    },
    [formatted, finish],
  );

  const handleNudge = useCallback(
    (direction: 1 | -1, big: boolean) => {
      const el = inputRef.current;
      if (el == null) return;
      const parsed = parseTimeStamp(text, anchors);
      if (!parsed.ok) return;
      const caret = el.selectionStart ?? text.length;
      const next = nudge(parsed.value, unitAt(caret), direction * (big ? 10 : 1));
      caretRef.current = caret;
      setText(formatTimeStamp(next));
    },
    [text, anchors, setText],
  );

  useLayoutEffect(() => {
    const caret = caretRef.current;
    if (caret == null || inputRef.current == null) return;
    caretRef.current = null;
    inputRef.current.setSelectionRange(caret, caret);
  }, [text]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") finish();
      else if (e.key === "Escape") {
        revertRef.current = true;
        finish();
      } else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
        e.preventDefault();
        const up = e.key === "ArrowUp";
        // Digits in the fixed layout nudge; a phrase walks its readings.
        if (FIXED_LAYOUT_RE.test(text.trim())) handleNudge(up ? 1 : -1, e.shiftKey);
        else
          setSelected((i) =>
            Math.max(0, Math.min(suggestions.length - 1, i + (up ? -1 : 1))),
          );
      }
    },
    [handleNudge, finish, text, suggestions.length],
  );

  const apply = useCallback(
    (next: number) => {
      revertRef.current = true;
      commit(next);
      setOpen(false);
    },
    [commit],
  );

  const parent = propsAnchors?.parent;

  const handleSuggestion = useCallback(
    (key: string) => {
      const hit = suggestions.find((sg) => sg.key === key);
      if (hit != null) apply(Number(hit.value.valueOf()));
    },
    [suggestions, apply],
  );

  const handleAction = useMemo(
    () => ({
      now: () => apply(TimeStamp.now().nanoseconds),
      empty: () => {
        if (emptyValue != null) apply(emptyValue);
      },
      parentStart: () => {
        if (parent != null) apply(parent.start);
      },
      parentEnd: () => {
        if (parent != null) apply(parent.end);
      },
    }),
    [apply, emptyValue, parent],
  );

  let label: ReactNode;
  if (stamp == null)
    label = <span className={CSS.BE("datetime", "empty")}>{placeholder}</span>;
  else {
    const showDay = sharedDay == null || !sameDay(stamp, new TimeStamp(sharedDay));
    label = (
      <>
        {showDay && (
          <span className={CSS.BE("datetime", "day")}>
            {describeDay(stamp, anchors.now)}
          </span>
        )}
        <span>{formatTime(stamp, resolution)}</span>
      </>
    );
  }

  return (
    <Cell
      label={label}
      open={open}
      onOpenChange={handleOpenChange}
      variant={variant}
      level={level}
      size={size}
      disabled={disabled}
      preview={preview}
      tooltip={stamp == null ? undefined : `${formatted} ${zoneName(stamp)}`}
      className={CSS.cls(CSS.B("datetime"), className)}
      field={
        <Text
          ref={inputRef}
          type="text"
          flush
          autoFocus
          rounded
          full="x"
          size="medium"
          value={text}
          onChange={setText}
          onKeyDown={handleKeyDown}
          onFocus={(e) => e.currentTarget.select()}
          placeholder="14:05, tomorrow 3pm, now - 5m"
          spellCheck={false}
        />
      }
    >
      <Flex.Box y gap="tiny" className={CSS.BE("datetime", "suggestions")}>
        <Suggestions
          text={text}
          suggestions={suggestions}
          selected={chosen?.key}
          anchors={anchors}
          onSelect={handleSuggestion}
        />
      </Flex.Box>
      <Flex.Box y gap="tiny" className={CSS.BE("datetime", "actions")}>
        <Menu.Menu onChange={handleAction}>
          <Action itemKey="now" icon={<Icon.Time />} label="Now" hint="now" />
          {emptyValue != null && !isEmpty && (
            <Action
              itemKey="empty"
              icon={<Icon.Close />}
              label={clearLabel}
              hint="empty"
            />
          )}
          {parent != null && (
            <Action
              itemKey="parentStart"
              icon={<Icon.Range />}
              label="Parent start"
              hint="T+0"
            />
          )}
          {parent != null && parent.end < TimeStamp.MAX.nanoseconds && (
            <Action
              itemKey="parentEnd"
              icon={<Icon.Range />}
              label="Parent end"
              hint="T+end"
            />
          )}
        </Menu.Menu>
      </Flex.Box>
    </Cell>
  );
};
