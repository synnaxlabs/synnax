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
  type CSSProperties,
  type KeyboardEvent,
  type ReactElement,
  useCallback,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { CSS as PCSS } from "@/css";
import { useCombinedRefs, useSyncedRef } from "@/hooks/ref";
import { type Input } from "@/input";
import { type state } from "@/state";
import { Text, type TextProps } from "@/text/Text";
import { triggerReflow } from "@/util/reflow";

/** Props for {@link Editable}. */
export type EditableProps = Omit<TextProps<"p">, "children" | "onChange"> &
  Input.Control<string> & {
    /** Lifts the editing flag out of the component so a parent can drive it. */
    useEditableState?: state.PureUse<boolean>;
    /** Whether a double click starts editing. Defaults to true. */
    allowDoubleClick?: boolean;
    /** Whether an empty value commits. It reverts otherwise. Defaults to false. */
    allowEmpty?: boolean;
    /** Whether to outline the field while editing. Defaults to true. */
    outline?: boolean;
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
      el.dispatchEvent(new Event(START_EDITING_EVENT_NAME));
      el.setAttribute("contenteditable", "true");
      el.addEventListener(RENAMED_EVENT_NAME, (e) =>
        resolve([getInnerText(e.target as HTMLElement), true]),
      );
      el.addEventListener(ESCAPED_EVENT_NAME, (e) =>
        resolve([getInnerText(e.target as HTMLElement), false]),
      );
    };
    tryEdit();
  });

/** Starts editing the {@link Editable} carrying the given id, discarding the result. */
export const edit = (id: string): void => {
  asyncEdit(id).catch(console.error);
};

const getInnerText = (el: HTMLElement): string => el.innerText.trim();

interface StylesToTriggerReflow {
  maxInlineSize?: CSSProperties["maxInlineSize"];
}

const compareStylesToTriggerReflow = (
  a: StylesToTriggerReflow | undefined,
  b: StylesToTriggerReflow | undefined,
): boolean => {
  if (a == null || b == null) return false;
  return a.maxInlineSize === b.maxInlineSize;
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
  useEditableState = useState,
  allowDoubleClick = true,
  onDoubleClick,
  allowEmpty = false,
  style,
  outline = true,
  ...rest
}: EditableProps): ReactElement => {
  const [editable, setEditable] = useEditableState(false);
  const ref = useRef<HTMLElement>(null);
  // Sometimes the onBlur event fires right after the user hits the enter key (since we
  // trigger it artificially). We track this value as an optimistic update to make sure
  // we don't call onChange twice in quick succession.
  const optimisticValueRef = useSyncedRef(value);

  // Turns out the writing modes like vertical-rl cause all sorts of problems with
  // elements whose values change. The following section of code forces the browser
  // to reflow the element when the value changes or the styles that affect the
  // layout change.
  const stylesToTriggerReflow = useRef<StylesToTriggerReflow | undefined>(style);
  const valueRef = useRef(value);
  if (
    (stylesToTriggerReflow.current != null &&
      !compareStylesToTriggerReflow(style, stylesToTriggerReflow.current)) ||
    value !== valueRef.current
  ) {
    triggerReflow(ref.current as HTMLElement);
    stylesToTriggerReflow.current = style;
    valueRef.current = value;
  }

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
    )
      return;
    if (forceEscape || (innerText.length === 0 && !allowEmpty)) {
      el.innerText = value;
      el.dispatchEvent(new Event(ESCAPED_EVENT_NAME));
    } else {
      onChange?.(innerText);
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
    el.blur();
  };

  const handleKeyUp = (e: KeyboardEvent<HTMLDivElement>): void => e.preventDefault();

  useLayoutEffect(() => {
    if (ref.current == null || !editable) return;
    triggerReflow(ref.current);
    const { current: el } = ref;
    el.focus();
    const range = document.createRange();
    range.selectNodeContents(el);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
  }, [editable]);

  if (ref.current !== null && !editable) ref.current.innerHTML = value;

  const refCallback = useCallback((el: HTMLElement) => {
    if (el == null) return;
    el.addEventListener(START_EDITING_EVENT_NAME, () => setEditable(true));
  }, []);

  const combinedRef = useCombinedRefs(propsRef, ref, refCallback);

  return (
    <Text
      ref={combinedRef}
      className={PCSS(
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
      onDoubleClick={handleDoubleClick}
      contentEditable={editable}
      suppressContentEditableWarning
      style={style}
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
  ...rest
}: MaybeEditableProps): ReactElement => {
  if (disabled || onChange == null || onChange === false)
    return <Text {...rest}>{value}</Text>;

  if (onChange === true) onChange = () => {};
  return (
    <Editable
      allowDoubleClick={allowDoubleClick}
      onChange={onChange}
      value={value}
      {...rest}
    />
  );
};
