// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { state } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useMemo, useState } from "react";
import { flushSync } from "react-dom";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { type Icon } from "@/icon";
import { Text } from "@/input/Text";
import { Effect } from "@/input/time/Effect";
import { type Suggestion } from "@/input/time/suggest";
import { type Variant } from "@/input/types";
import { Menu } from "@/menu";
import { Text as BaseText } from "@/text";

/** The props a time input forwards to its trigger, minus the ones it owns. */
export interface BaseProps extends Omit<
  Dialog.TriggerProps,
  "value" | "onChange" | "children" | "variant" | "hideCaret"
> {
  variant?: Variant;
}

/** A fixed value the editor offers under the readings. */
export interface Action<V> {
  key: string;
  icon: Icon.ReactElement;
  label: string;
  /** An expression that types the same value, shown when it differs from the label. */
  hint?: string;
  /** Called on render and again on a click, so a value like now stays current. */
  value: () => V;
}

export interface EditorProps<V> extends BaseProps {
  /** The value as content, shown at rest. */
  label: ReactNode;
  /** The text the editor opens on. */
  initialText: string;
  /** Lists the ways the text can be read, most likely first. */
  suggest: (text: string) => Suggestion<V>[];
  /** Called with the reading a commit takes. */
  onCommit: (value: V) => void;
  /** Called when a commit finds the field blank. */
  onClear?: () => void;
  /**
   * Moves the unit under the caret by `steps`.
   * @returns the new text, or null when the text has no units; the arrow keys then
   * walk the readings.
   */
  nudge?: (text: string, caret: number, steps: number) => string | null;
  /** Shown while the field is blank. */
  hint: string;
  /** Shown when the text has no reading. */
  unreadMessage: string;
  fieldPlaceholder?: string;
  actions?: Action<V>[];
  /**
   * Rendered under the options with the value the highlighted reading or the hovered
   * action would commit. Use it to say what else a commit would change, and return
   * null when nothing else would.
   */
  effect?: Component.RenderProp<{ candidate: V }>;
  /** Renders one reading on one line. */
  children: Component.RenderProp<Suggestion<V>>;
}

/** An open editor's state. The editor is closed while it has none. */
interface Draft {
  text: string;
  /** The index of the highlighted reading. */
  selected: number;
  /** The key of the action under the pointer or focus. */
  hoveredAction: string | null;
}

const openDraft = (text: string): Draft => ({ text, selected: 0, hoveredAction: null });

/**
 * A value that reads as content and edits in a connected dialog under it, the shape
 * of a select. The dialog holds a text field over the readings of its text. Enter or a
 * click outside commits the highlighted reading; Escape discards the edit; Enter on
 * text with no reading keeps the editor open. Hovering a reading highlights it.
 */
export const Editor = <V,>({
  label,
  initialText,
  suggest,
  onCommit,
  onClear,
  nudge,
  hint,
  unreadMessage,
  fieldPlaceholder,
  actions,
  effect,
  children,
  variant = "outlined",
  className,
  style,
  tooltip,
  ...rest
}: EditorProps<V>): ReactElement => {
  const [draft, setDraft] = useState<Draft | null>(null);
  const text = draft?.text;
  const suggestions = useMemo(
    () => (text == null ? [] : suggest(text)),
    [suggest, text],
  );
  const selected = draft?.selected ?? 0;
  const chosen = suggestions[Math.min(selected, suggestions.length - 1)];
  const blank = text == null || text.trim().length === 0;
  const unread = !blank && suggestions.length === 0;

  const update = (patch: Partial<Draft>): void =>
    setDraft((prev) => (prev == null ? prev : { ...prev, ...patch }));
  const setText = (next: string): void => update({ text: next, selected: 0 });
  const close = (): void => setDraft(null);

  const commit = (value: V): void => {
    onCommit(value);
    close();
  };

  const finish = (): void => {
    if (blank) onClear?.();
    else if (chosen != null) onCommit(chosen.value);
    close();
  };

  const handleVisibleChange = (next: state.SetArg<boolean>): void => {
    if (state.executeSetter(next, draft != null)) setDraft(openDraft(initialText));
    else finish();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "Enter") {
      if (!unread) finish();
    } else if (e.key === "Escape") close();
    else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
      e.preventDefault();
      const up = e.key === "ArrowUp";
      const field = e.currentTarget;
      const current = field.value;
      const caret = field.selectionStart ?? current.length;
      const nudged = nudge?.(current, caret, (up ? 1 : -1) * (e.shiftKey ? 10 : 1));
      if (nudged == null) {
        const last = suggestions.length - 1;
        const step = up ? -1 : 1;
        return setDraft(
          (prev) =>
            prev && {
              ...prev,
              selected: Math.max(0, Math.min(last, prev.selected + step)),
            },
        );
      }
      // The field re-renders with the new text before the caret moves back into it.
      flushSync(() => setText(nudged));
      field.setSelectionRange(caret, caret);
    }
  };

  const handleSuggestion = (key: string): void => {
    const hit = suggestions.find((sg) => sg.key === key);
    if (hit != null) commit(hit.value);
  };

  const handleAction = (key: string): void => {
    const hit = actions?.find((a) => a.key === key);
    if (hit != null) commit(hit.value());
  };

  const action = actions?.find((a) => a.key === draft?.hoveredAction);
  const candidate = action != null ? action.value() : chosen?.value;

  return (
    <Dialog.Frame
      variant="connected"
      visible={draft != null}
      onVisibleChange={handleVisibleChange}
      className={CSS.cls(CSS.B("time-editor"), CSS.M(variant), className)}
      style={style}
    >
      <Dialog.Trigger
        hideCaret
        variant={variant === "outlined" ? "outlined" : "text"}
        tooltip={draft != null ? undefined : tooltip}
        tooltipLocation="bottom"
        className={CSS.BE("time-editor", "trigger")}
        {...rest}
      >
        {label}
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("time-editor", "dialog")} bordered={false}>
        <Text
          type="text"
          flush
          autoFocus
          rounded
          full="x"
          size="medium"
          value={text ?? ""}
          onChange={setText}
          onKeyDown={handleKeyDown}
          onFocus={(e) => e.currentTarget.select()}
          placeholder={fieldPlaceholder}
          spellCheck={false}
        />
        <Flex.Box y empty bordered borderColor={6} rounded full="x">
          <Flex.Box y gap="tiny" className={CSS.BE("time-editor", "suggestions")}>
            {blank && (
              <BaseText.Text
                level="small"
                color={9}
                className={CSS.BE("time-editor", "note")}
              >
                {hint}
              </BaseText.Text>
            )}
            {unread && (
              <BaseText.Text
                level="small"
                status="error"
                className={CSS.BE("time-editor", "note")}
              >
                {unreadMessage}
              </BaseText.Text>
            )}
            {suggestions.length > 0 && (
              <Menu.Menu value={chosen?.key} onChange={handleSuggestion}>
                {suggestions.map((suggestion, i) => (
                  <Menu.Item
                    key={suggestion.key}
                    itemKey={suggestion.key}
                    className={CSS.BE("time-editor", "suggestion")}
                    onMouseEnter={() => update({ selected: i })}
                  >
                    {children(suggestion)}
                  </Menu.Item>
                ))}
              </Menu.Menu>
            )}
          </Flex.Box>
          {actions != null && actions.length > 0 && (
            <Flex.Box y gap="tiny" className={CSS.BE("time-editor", "actions")}>
              <Menu.Menu onChange={handleAction}>
                {actions.map(({ key, icon, label, hint }) => (
                  <Menu.Item
                    key={key}
                    itemKey={key}
                    className={CSS.BE("time-editor", "action")}
                    onMouseEnter={() => update({ hoveredAction: key })}
                    onMouseLeave={() => update({ hoveredAction: null })}
                    onFocus={() => update({ hoveredAction: key })}
                    onBlur={() => update({ hoveredAction: null })}
                  >
                    {icon}
                    {label}
                    {hint != null && (
                      <BaseText.Text
                        color={9}
                        className={CSS.BE("time-editor", "hint")}
                      >
                        {hint}
                      </BaseText.Text>
                    )}
                  </Menu.Item>
                ))}
              </Menu.Menu>
            </Flex.Box>
          )}
          {effect != null && (
            <Effect>{candidate == null ? null : effect({ candidate })}</Effect>
          )}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
