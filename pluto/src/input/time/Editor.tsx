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
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { type Icon } from "@/icon";
import { Text } from "@/input/Text";
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
  /** The expression that types the same value. */
  hint: string;
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
   * Rendered once in every reading and action with the value it would commit. Use it
   * to say what else a commit would change.
   */
  effect?: Component.RenderProp<{ candidate: V }>;
  /** Renders one reading. */
  children: Component.RenderProp<Suggestion<V>>;
}

/**
 * A value that reads as content and edits in a connected dialog under it, the shape
 * of a select. The dialog holds a text field over the readings of its text. Enter or a
 * click outside commits the highlighted reading; Escape discards the edit; Enter on
 * text with no reading keeps the editor open.
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
  const [visible, setVisible] = useState(false);
  const [text, setTextState] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const caretRef = useRef<number | null>(null);

  const setText = useCallback((next: string) => {
    setTextState(next);
    setSelected(0);
  }, []);

  const suggestions = useMemo(() => suggest(text), [suggest, text]);
  const chosen = suggestions[Math.min(selected, suggestions.length - 1)];
  const blank = text.trim().length === 0;
  const unread = !blank && suggestions.length === 0;

  const close = useCallback(() => setVisible(false), []);

  const commit = useCallback(
    (value: V) => {
      onCommit(value);
      close();
    },
    [onCommit, close],
  );

  const finish = useCallback(() => {
    if (blank) onClear?.();
    else if (chosen != null) onCommit(chosen.value);
    close();
  }, [blank, chosen, onClear, onCommit, close]);

  const handleVisibleChange = useCallback(
    (next: state.SetArg<boolean>) => {
      if (!state.executeSetter(next, visible)) return finish();
      setText(initialText);
      setVisible(true);
    },
    [visible, finish, setText, initialText],
  );

  useLayoutEffect(() => {
    const caret = caretRef.current;
    if (caret == null || inputRef.current == null) return;
    caretRef.current = null;
    inputRef.current.setSelectionRange(caret, caret);
  }, [text]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        if (!unread) finish();
      } else if (e.key === "Escape") close();
      else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
        e.preventDefault();
        const up = e.key === "ArrowUp";
        const caret = e.currentTarget.selectionStart ?? text.length;
        const nudged = nudge?.(text, caret, (up ? 1 : -1) * (e.shiftKey ? 10 : 1));
        if (nudged != null) {
          caretRef.current = caret;
          return setText(nudged);
        }
        setSelected((i) =>
          Math.max(0, Math.min(suggestions.length - 1, i + (up ? -1 : 1))),
        );
      }
    },
    [unread, finish, close, nudge, text, setText, suggestions.length],
  );

  const handleSuggestion = useCallback(
    (key: string) => {
      const hit = suggestions.find((sg) => sg.key === key);
      if (hit != null) commit(hit.value);
    },
    [suggestions, commit],
  );

  const handleAction = useCallback(
    (key: string) => {
      const hit = actions?.find((a) => a.key === key);
      if (hit != null) commit(hit.value());
    },
    [actions, commit],
  );

  // Caller content sits in its own box, so it never displaces the hints beside it.
  const renderEffect = (candidate: V): ReactElement | null =>
    effect == null ? null : (
      <Flex.Box
        x
        gap="small"
        align="center"
        className={CSS.BE("time-editor", "effect")}
      >
        {effect({ candidate })}
      </Flex.Box>
    );

  return (
    <Dialog.Frame
      variant="connected"
      visible={visible}
      onVisibleChange={handleVisibleChange}
      className={CSS.cls(CSS.B("time-editor"), CSS.M(variant), className)}
      style={style}
    >
      <Dialog.Trigger
        hideCaret
        variant={variant === "outlined" ? "outlined" : "text"}
        tooltip={visible ? undefined : tooltip}
        tooltipLocation="bottom"
        className={CSS.BE("time-editor", "trigger")}
        {...rest}
      >
        {label}
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("time-editor", "dialog")} bordered={false}>
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
                {suggestions.map((suggestion) => (
                  <Menu.Item
                    key={suggestion.key}
                    itemKey={suggestion.key}
                    className={CSS.BE("time-editor", "suggestion")}
                  >
                    <Flex.Box y gap="tiny" grow>
                      {children(suggestion)}
                      {renderEffect(suggestion.value)}
                    </Flex.Box>
                  </Menu.Item>
                ))}
              </Menu.Menu>
            )}
          </Flex.Box>
          {actions != null && actions.length > 0 && (
            <Flex.Box y gap="tiny" className={CSS.BE("time-editor", "actions")}>
              <Menu.Menu onChange={handleAction}>
                {actions.map(({ key, icon, label, hint, value }) => (
                  <Menu.Item
                    key={key}
                    itemKey={key}
                    className={CSS.BE("time-editor", "action")}
                  >
                    {icon}
                    {label}
                    <BaseText.Text color={9} className={CSS.BE("time-editor", "hint")}>
                      {hint}
                    </BaseText.Text>
                    {renderEffect(value())}
                  </Menu.Item>
                ))}
              </Menu.Menu>
            </Flex.Box>
          )}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
