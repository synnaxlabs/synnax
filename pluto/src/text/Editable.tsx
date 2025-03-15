// Copyright 2025 Synnax Labs, Inc.
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
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { type Input } from "@/input";
import { type state } from "@/state";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";
import { triggerReflow } from "@/util/reflow";

export type EditableProps<L extends text.Level = "h1"> = Omit<
  TextProps<L>,
  "children" | "onChange"
> &
  Input.Control<string> & {
    useEditableState?: state.PureUse<boolean>;
    allowDoubleClick?: boolean;
    allowEmpty?: boolean;
    outline?: boolean;
  };

const NOMINAL_EXIT_KEYS = ["Escape", "Enter"];
const BASE_CLASS = CSS.BM("text", "editable");
const MAX_EDIT_RETRIES = 10;
const RENAMED_EVENT_NAME = "renamed";
const ESCAPED_EVENT_NAME = "escaped";

export const edit = (
  id: string,
  onChange?: (value: string, renamed: boolean) => void,
): void => {
  let currRetry = 0;
  const tryEdit = (): void => {
    currRetry++;
    const d = document.getElementById(id);
    if (d == null || !d.classList.contains(BASE_CLASS)) {
      if (currRetry < MAX_EDIT_RETRIES) setTimeout(() => tryEdit(), 100);
      else throw new Error(`Could not find element with id ${id}`);
      return;
    }
    d.setAttribute("contenteditable", "true");
    if (onChange == null) return;
    d.addEventListener(RENAMED_EVENT_NAME, (e) =>
      onChange(getInnerText(e.target as HTMLElement), true),
    );
    d.addEventListener(ESCAPED_EVENT_NAME, (e) =>
      onChange(getInnerText(e.target as HTMLElement), false),
    );
  };
  tryEdit();
};

export const asyncEdit = (id: string): Promise<[string, boolean]> =>
  new Promise((resolve) => {
    const onChange = (value: string, renamed: boolean): void =>
      resolve([value, renamed]);
    edit(id, onChange);
  });

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

export const Editable = <L extends text.Level = text.Level>({
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
}: EditableProps<L>): ReactElement => {
  const [editable, setEditable] = useEditableState(false);
  const ref = useRef<HTMLElement>(null);
  // Sometimes the onBlur event fires right after the user hits
  // the enter key (since we trigger it artificially). We track
  // this value as an optimistic update to make sure we don't
  // call onChange twice in quick succession.
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

  const handleDoubleClick = (
    e: React.MouseEvent<HTMLParagraphElement, MouseEvent>,
  ): void => {
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

  const handleKeyDown = (e: KeyboardEvent<HTMLParagraphElement>): void => {
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

  useEffect(() => {
    if (ref.current == null) return;
    const m = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName !== "contenteditable") return;
        const t = mutation.target as HTMLElement;
        const makeEditable = t.contentEditable === "true";
        if (makeEditable) setEditable(true);
      });
    });
    m.observe(ref.current as Node, { attributes: true });
  }, []);

  return (
    // @ts-expect-error - TODO: generic element behavior is funky
    <Text<L>
      ref={ref}
      className={CSS(
        className,
        CSS.BM("text", "editable"),
        outline && CSS.M("outline"),
      )}
      onBlur={() => {
        setEditable(false);
        const el = ref.current;
        if (el == null) return;
        handleUpdate(el);
      }}
      onKeyDown={handleKeyDown}
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

export type MaybeEditableProps<L extends text.Level = "h1"> = Omit<
  EditableProps<L>,
  "onChange"
> & {
  onChange?: EditableProps<L>["onChange"] | boolean;
  disabled?: boolean;
};

export const MaybeEditable = <L extends text.Level = text.Level>({
  onChange,
  disabled = false,
  value,
  allowDoubleClick,
  ...rest
}: MaybeEditableProps<L>): ReactElement => {
  if (disabled || onChange == null || typeof onChange === "boolean")
    // @ts-expect-error - generic component errors
    return <Text<L> {...rest}>{value}</Text>;

  return (
    <>
      {/* @ts-expect-error - generic component errors */}
      <Editable<L>
        allowDoubleClick={allowDoubleClick}
        onChange={onChange}
        value={value}
        {...rest}
      />
    </>
  );
};
