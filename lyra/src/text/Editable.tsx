// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Editable.css";

import {
  type KeyboardEvent,
  type ReactElement,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { CSS as PCSS } from "@/css";
import { useCombinedRefs, useSyncedRef } from "@/hooks/ref";
import { type Input } from "@/input";
import { paintHighlight } from "@/text/highlight";
import { Text, type TextProps } from "@/text/Text";
import { triggerReflow } from "@/util/reflow";

/** Props for {@link Editable}. */
export type EditableProps = Omit<TextProps<"p">, "children" | "onChange"> &
  Omit<Input.Control<string>, "onChange"> & {
    /** Receives the committed text. Return false to reject it, reverting the field to
     * value; the field otherwise holds the committed text until value catches up. */
    onChange: (value: string) => void | boolean;
    /** Whether a double click starts editing. Defaults to true. */
    allowDoubleClick?: boolean;
    /** Whether an empty value commits. It reverts otherwise. Defaults to false. */
    allowEmpty?: boolean;
    /** Whether to outline the field while editing. Defaults to true. */
    outline?: boolean;
    /** The most characters the field takes. Typed or pasted text past it is dropped. */
    maxLength?: number;
  };

const NOMINAL_EXIT_KEYS = ["Escape", "Enter"];
const BASE_CLASS = PCSS.BM("text", "editable");
const MAX_EDIT_RETRIES = 10;
const RENAMED_EVENT_NAME = "renamed";
const ESCAPED_EVENT_NAME = "escaped";
const START_EDITING_EVENT_NAME = "start-editing";

// Read-only copies of the same value render with the same id, so the id alone does not
// identify the edit target. Selecting on the class first and matching the id by hand
// keeps every candidate in play: a `#id.class` selector resolves the id to one element
// before testing the class, and gives up when that element is a read-only copy.
const findEditable = (id: string): Element | undefined =>
  Array.from(document.getElementsByClassName(BASE_CLASS)).find((el) => el.id === id);

/**
 * Starts editing the {@link Editable} carrying the given id, retrying for a second
 * while it mounts. Use it to rename a resource the same gesture just created.
 *
 * @returns the final text and whether the user committed it. A rejected promise means
 * no such element appeared.
 */
export const asyncEdit = (id: string): Promise<[string, boolean]> =>
  new Promise((resolve, reject) => {
    let currRetry = 0;
    const tryEdit = (): void => {
      currRetry++;
      const el = findEditable(id);
      if (el == null) {
        if (currRetry < MAX_EDIT_RETRIES) setTimeout(tryEdit, 100);
        else reject(new Error(`Could not find element with id ${id}`));
        return;
      }
      el.dispatchEvent(new CustomEvent(START_EDITING_EVENT_NAME));
      el.setAttribute("contenteditable", "true");
      const handleRenamed = (e: Event): void => {
        el.removeEventListener(ESCAPED_EVENT_NAME, handleEscaped);
        resolve([getInnerText(e.target as HTMLElement), true]);
      };
      const handleEscaped = (e: Event): void => {
        el.removeEventListener(RENAMED_EVENT_NAME, handleRenamed);
        resolve([getInnerText(e.target as HTMLElement), false]);
      };
      el.addEventListener(RENAMED_EVENT_NAME, handleRenamed, { once: true });
      el.addEventListener(ESCAPED_EVENT_NAME, handleEscaped, { once: true });
    };
    tryEdit();
  });

/** Starts editing the {@link Editable} carrying the given id, discarding the result. */
export const edit = (id: string): void => {
  asyncEdit(id).catch(console.error);
};

const getInnerText = (el: HTMLElement): string => el.innerText.trim();

const HIGH_SURROGATE = /[\uD800-\uDBFF]/;

// Input lands before the caret, so the characters past the limit are the ones just
// before it.
const dropOverflow = (el: HTMLElement, maxLength: number): void => {
  const text = el.textContent;
  const overflow = text.length - maxLength;
  if (overflow <= 0) return;
  const selection = window.getSelection();
  let caret = text.length;
  if (selection?.focusNode != null && el.contains(selection.focusNode)) {
    // Pasted markup can split the text, so the caret's offset spans every child node.
    const before = document.createRange();
    before.selectNodeContents(el);
    before.setEnd(selection.focusNode, selection.focusOffset);
    caret = before.toString().length;
  }
  const end = Math.max(caret, overflow);
  let kept = end - overflow;
  // Keep a surrogate pair whole, or the cut leaves half of an emoji.
  if (kept > 0 && HIGH_SURROGATE.test(text[kept - 1])) kept--;
  el.textContent = text.slice(0, kept) + text.slice(end);
  if (el.firstChild != null) selection?.collapse(el.firstChild, kept);
};

/**
 * Text that turns into a field in place. Commits on Enter or blur, reverts on Escape.
 * Give it an id to drive it from elsewhere with {@link edit}.
 */
export const Editable = ({
  ref: propsRef,
  onChange,
  value,
  className,
  allowDoubleClick = true,
  onDoubleClick,
  allowEmpty = false,
  style,
  outline = true,
  maxLength,
  onInput,
  highlight,
  ...rest
}: EditableProps): ReactElement => {
  const [editable, setEditable] = useState(false);
  const ref = useRef<HTMLElement>(null);
  // Sometimes the onBlur event fires right after the user hits the enter key (since we
  // trigger it artificially). We track this value as an optimistic update to make sure
  // we don't call onChange twice in quick succession.
  const optimisticValueRef = useSyncedRef(value);
  // Holds the text this field committed and the value it replaced. Exiting an edit
  // repaints from value, which still carries the old text until the caller's write
  // lands, so the committed text stands in until value moves off what it replaced.
  const committedRef = useRef<{ text: string; replaced: string } | null>(null);
  // Enter and Escape give the focus back here, such as to a list that answers the arrow
  // keys.
  const returnFocusRef = useRef<HTMLElement | null>(null);

  // Writing modes like vertical-rl do not lay out again when the value or the inline
  // size changes, so a change forces a reflow.
  const maxInlineSize = style?.maxInlineSize;
  const reflowedRef = useRef({ value, maxInlineSize });
  useLayoutEffect(() => {
    const prev = reflowedRef.current;
    if (prev.value === value && prev.maxInlineSize === maxInlineSize) return;
    reflowedRef.current = { value, maxInlineSize };
    if (ref.current != null) triggerReflow(ref.current);
  }, [value, maxInlineSize]);

  const handleDoubleClick = (e: React.MouseEvent<HTMLDivElement>): void => {
    if (allowDoubleClick) {
      setEditable(true);
      triggerReflow(ref.current as HTMLElement);
    }
    onDoubleClick?.(e);
  };

  const handleUpdate = (el: HTMLElement, forceEscape = false): void => {
    const innerText = getInnerText(el);
    if (
      optimisticValueRef.current === innerText &&
      (innerText.length > 0 || allowEmpty)
    ) {
      // An unchanged commit is a cancel: without the event, an awaiting asyncEdit
      // caller would hang forever.
      el.dispatchEvent(new Event(ESCAPED_EVENT_NAME));
      return;
    }
    if (forceEscape || (innerText.length === 0 && !allowEmpty)) {
      committedRef.current = null;
      el.innerText = value;
      el.dispatchEvent(new Event(ESCAPED_EVENT_NAME));
    } else {
      committedRef.current = { text: innerText, replaced: value };
      if (onChange?.(innerText) === false) {
        committedRef.current = null;
        el.innerText = value;
        el.dispatchEvent(new Event(ESCAPED_EVENT_NAME));
        return;
      }
      optimisticValueRef.current = innerText;
      el.dispatchEvent(new Event(RENAMED_EVENT_NAME));
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    if (ref.current == null) return;
    triggerReflow(ref.current);
    if (!editable || !NOMINAL_EXIT_KEYS.includes(e.key)) return;
    e.stopPropagation();
    e.preventDefault();
    const el = ref.current;
    if (ref.current == null) return;
    setEditable(false);
    handleUpdate(el, e.key === "Escape");
    returnFocusRef.current?.focus({ preventScroll: true });
    if (document.activeElement === el) el.blur();
  };

  const handleKeyUp = (e: KeyboardEvent<HTMLDivElement>): void => e.preventDefault();

  useLayoutEffect(() => {
    if (ref.current == null || !editable) return;
    triggerReflow(ref.current);
    const { current: el } = ref;
    // An edit runs over plain text, so the tint marks go.
    el.replaceChildren(el.textContent);
    const active = document.activeElement;
    returnFocusRef.current = active instanceof HTMLElement ? active : null;
    el.focus();
    const range = document.createRange();
    range.selectNodeContents(el);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
    // Editing scrolls a cut field to the caret, so the end of an edit scrolls it back.
    // Not scrollTo: WebKit ignores it until the field lays out again after a reflow.
    return () => {
      el.scrollLeft = 0;
    };
  }, [editable]);

  // Editing replaces the text React rendered, so the shown text is written by hand.
  useLayoutEffect(() => {
    if (ref.current == null || editable) return;
    const { current: committed } = committedRef;
    if (committed != null && value === committed.replaced)
      paintHighlight(ref.current, committed.text, highlight);
    else {
      committedRef.current = null;
      paintHighlight(ref.current, value, highlight);
    }
  });

  // A ref callback, so each element Text renders listens before asyncEdit can find it.
  const listenForEdits = (el: HTMLElement | null) => {
    if (el == null) return;
    const handleStartEditing = (): void => setEditable(true);
    el.addEventListener(START_EDITING_EVENT_NAME, handleStartEditing);
    return () => el.removeEventListener(START_EDITING_EVENT_NAME, handleStartEditing);
  };

  const combinedRef = useCombinedRefs(propsRef, ref, listenForEdits);

  return (
    <Text
      ref={combinedRef}
      className={PCSS.cls(
        className,
        PCSS.BM("text", "editable"),
        outline && PCSS.M("outline"),
      )}
      onBlur={() => {
        setEditable(false);
        const el = ref.current;
        if (el == null) return;
        handleUpdate(el);
      }}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
      onInput={(e) => {
        if (maxLength != null && !e.nativeEvent.isComposing)
          dropOverflow(e.currentTarget, maxLength);
        onInput?.(e);
      }}
      onDoubleClick={handleDoubleClick}
      contentEditable={editable}
      suppressContentEditableWarning
      style={style}
      // Text squares a lone glyph as an icon. Editable text is a name, never an icon.
      square={false}
      {...rest}
    >
      {value}
    </Text>
  );
};

/** Props for {@link MaybeEditable}. */
export type MaybeEditableProps = Omit<EditableProps, "onChange"> & {
  /** A handler makes the text editable. true makes it editable but discards the edit;
   * false or absent renders plain text. */
  onChange?: EditableProps["onChange"] | boolean;
  /** Forces plain text whatever onChange says. */
  disabled?: boolean;
};

/**
 * Renders {@link Editable} when the caller can accept an edit and plain {@link Text}
 * when it cannot, so a caller gated on a permission needs no branch of its own.
 */
export const MaybeEditable = ({
  onChange,
  disabled = false,
  value,
  allowDoubleClick,
  maxLength,
  ...rest
}: MaybeEditableProps): ReactElement => {
  if (disabled || onChange == null || onChange === false)
    return (
      <Text square={false} {...rest}>
        {value}
      </Text>
    );

  if (onChange === true) onChange = () => {};
  return (
    <Editable
      allowDoubleClick={allowDoubleClick}
      maxLength={maxLength}
      onChange={onChange}
      value={value}
      {...rest}
    />
  );
};
