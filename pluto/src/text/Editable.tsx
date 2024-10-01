// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Editable.css";

import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { type Input } from "@/input";
import { type state } from "@/state";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type EditableProps<L extends text.Level = "h1"> = Omit<
  TextProps<L>,
  "children" | "onChange"
> &
  Input.Control<string> & {
    useEditableState?: state.PureUse<boolean>;
    allowDoubleClick?: boolean;
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

export const Editable = <L extends text.Level = text.Level>({
  onChange,
  value,
  useEditableState = useState,
  allowDoubleClick = true,
  onDoubleClick,
  ...props
}: EditableProps<L>): ReactElement => {
  const [editable, setEditable] = useEditableState(false);
  const ref = useRef<HTMLElement>(null);

  const handleDoubleClick = (
    e: React.MouseEvent<HTMLParagraphElement, MouseEvent>,
  ): void => {
    if (allowDoubleClick) setEditable(true);
    onDoubleClick?.(e);
  };

  const handleUpdate = (el: HTMLElement, forceEscape = false): void => {
    const innerText = getInnerText(el);
    if (forceEscape || innerText.length === 0) {
      el.innerText = value;
      el.dispatchEvent(new Event(ESCAPED_EVENT_NAME));
    } else {
      onChange?.(innerText);
      el.dispatchEvent(new Event(RENAMED_EVENT_NAME));
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLParagraphElement>): void => {
    if (!editable || !NOMINAL_EXIT_KEYS.includes(e.key) || ref.current == null) return;
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
      className={CSS.BM("text", "editable")}
      onBlur={() => {
        setEditable(false);
        const el = ref.current;
        if (el == null) return;
        handleUpdate(el);
      }}
      onKeyDown={handleKeyDown}
      onKeyUp={(e: KeyboardEvent<HTMLParagraphElement>) => {
        e.stopPropagation();
        e.preventDefault();
      }}
      onDoubleClick={handleDoubleClick}
      contentEditable={editable}
      suppressContentEditableWarning
      {...props}
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
  ...props
}: MaybeEditableProps<L>): ReactElement => {
  if (disabled || onChange == null || typeof onChange === "boolean")
    // @ts-expect-error - generic component errors
    return <Text<L> {...props}>{value}</Text>;

  return (
    <>
      {/* @ts-expect-error - generic component errors */}
      <Editable<L>
        allowDoubleClick={allowDoubleClick}
        onChange={onChange}
        value={value}
        {...props}
      />
    </>
  );
};
