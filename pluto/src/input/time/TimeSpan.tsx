// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { type text, TimeSpan as XTimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Text } from "@/input/Text";
import { Cell } from "@/input/time/Cell";
import { roundNumeric, truncateSpan } from "@/input/time/grammar";
import { suggestTimeSpans } from "@/input/time/suggest";
import { type Control, type Variant } from "@/input/types";
import { Menu } from "@/menu";
import { Text as BaseText } from "@/text";

export interface TimeSpanProps extends Control<number> {
  /**
   * When set, the cell shows the time elapsed since this instant, ticking every
   * second, instead of `value`. Typing a duration still commits through `onChange`.
   */
  elapsedSince?: number;
  /** The finest unit the label shows; the tooltip and editor keep every digit. */
  resolution?: XTimeSpan;
  variant?: Variant;
  level?: text.Level;
  size?: Component.Size;
  disabled?: boolean;
  preview?: boolean;
  className?: string;
}

/** Formats a span for the cell; zero reads as `0s`. */
export const formatTimeSpan = (span: XTimeSpan): string =>
  span.toString("full") || "0s";

const useElapsed = (
  since: number | undefined,
  resolution: XTimeSpan = XTimeSpan.SECOND,
): string | null => {
  const [, setTick] = useState(0);
  useEffect(() => {
    if (since == null) return;
    const i = setInterval(() => setTick((p) => p + 1), 1000);
    return () => clearInterval(i);
  }, [since]);
  if (since == null) return null;
  // The clock ticks by the second, so it never reads finer than that.
  const shown = resolution.lessThan(XTimeSpan.SECOND) ? XTimeSpan.SECOND : resolution;
  return formatTimeSpan(truncateSpan(TimeStamp.now().span(since), shown));
};

/**
 * A duration cell. The value is nanoseconds. At rest it reads `2h 30m 12s`; clicking
 * opens an editor that accepts any duration the grammar does (`30s`, `1:30:00`,
 * `1.5h`) and commits on Enter or a click outside.
 */
export const TimeSpan = ({
  value,
  onChange,
  elapsedSince,
  resolution,
  variant = "outlined",
  className,
  level,
  size,
  disabled,
  preview,
}: TimeSpanProps): ReactElement => {
  const [open, setOpen] = useState(false);
  const [text, setTextState] = useState("");
  const [selected, setSelected] = useState(0);
  const revertRef = useRef(false);
  const elapsed = useElapsed(elapsedSince, resolution);

  const exact = new XTimeSpan(roundNumeric(value));
  const formatted = formatTimeSpan(exact);
  const shown =
    resolution == null ? formatted : formatTimeSpan(truncateSpan(exact, resolution));

  const setText = useCallback((next: string) => {
    setTextState(next);
    setSelected(0);
  }, []);

  const suggestions = useMemo(() => suggestTimeSpans(text), [text]);
  const chosen = suggestions[Math.min(selected, suggestions.length - 1)];

  const apply = useCallback(
    (span: XTimeSpan) => {
      const next = Number(span.valueOf());
      // An elapsed readout ignores value, so any typed duration is a change.
      if (elapsedSince != null || next !== value) onChange(next);
    },
    [value, onChange, elapsedSince],
  );

  const finish = useCallback(() => {
    if (!revertRef.current && chosen != null) apply(chosen.value);
    revertRef.current = false;
    setOpen(false);
  }, [chosen, apply]);

  const handleSuggestion = useCallback(
    (key: string) => {
      const hit = suggestions.find((sg) => sg.key === key);
      if (hit == null) return;
      revertRef.current = true;
      apply(hit.value);
      setOpen(false);
    },
    [suggestions, apply],
  );

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (next) {
        // An elapsed readout is a clock, not a value; the user types a fresh duration.
        setText(elapsed == null ? formatted : "");
        setOpen(true);
      } else finish();
    },
    [elapsed, formatted, finish],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") finish();
      else if (e.key === "Escape") {
        revertRef.current = true;
        finish();
      } else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
        e.preventDefault();
        const step = e.key === "ArrowUp" ? -1 : 1;
        setSelected((i) => Math.max(0, Math.min(suggestions.length - 1, i + step)));
      }
    },
    [finish, suggestions.length],
  );

  const blank = text.trim().length === 0;
  const unread = !blank && suggestions.length === 0;

  return (
    <Cell
      label={<span className={CSS.BE("timespan", "label")}>{elapsed ?? shown}</span>}
      tooltip={elapsed == null && shown !== formatted ? formatted : undefined}
      open={open}
      onOpenChange={handleOpenChange}
      variant={variant}
      level={level}
      size={size}
      disabled={disabled}
      preview={preview}
      className={CSS.cls(
        CSS.B("timespan"),
        elapsed != null && CSS.M("elapsed"),
        className,
      )}
      field={
        <Text
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
          placeholder={elapsed ?? "2h 30m"}
          spellCheck={false}
        />
      }
    >
      <Flex.Box y gap="tiny" className={CSS.BE("timespan", "suggestions")}>
        {blank && (
          <BaseText.Text level="small" color={9} className={CSS.BE("timespan", "note")}>
            Try 2h 30m, 1:30:00, 90, or an hour and a half
          </BaseText.Text>
        )}
        {unread && (
          <BaseText.Text
            level="small"
            status="error"
            className={CSS.BE("timespan", "note")}
          >
            Not a duration
          </BaseText.Text>
        )}
        {!blank && !unread && (
          <Menu.Menu value={chosen?.key} onChange={handleSuggestion}>
            {suggestions.map(({ key, value: span, reading }) => (
              <Menu.Item
                key={key}
                itemKey={key}
                className={CSS.BE("timespan", "suggestion")}
              >
                <Flex.Box y gap="tiny" grow>
                  <span>{formatTimeSpan(span)}</span>
                  <BaseText.Text level="small" color={9} overflow="ellipsis">
                    {reading}
                  </BaseText.Text>
                </Flex.Box>
              </Menu.Item>
            ))}
          </Menu.Menu>
        )}
      </Flex.Box>
    </Cell>
  );
};
