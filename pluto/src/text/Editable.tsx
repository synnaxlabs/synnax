// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { KeyboardEvent, ReactElement } from "react";

import { CSS } from "@/css";
import { type Input } from "@/input";
import { type state } from "@/state";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

import "@/text/Editable.css";

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

export const edit = (id: string): void => {
  const d = document.getElementById(id);
  if (d == null || !d.classList.contains(BASE_CLASS))
    return console.error(`Element with id ${id} is not an instance of Text.Editable`);
  d.setAttribute("contenteditable", "true");
};

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

  const handleKeyDown = (e: KeyboardEvent<HTMLParagraphElement>): void => {
    if (!editable || !NOMINAL_EXIT_KEYS.includes(e.key) || ref.current == null) return;
    e.stopPropagation();
    e.preventDefault();
    const el = ref.current;
    setEditable(false);
    onChange?.(el.innerText.trim());
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
    m.observe(ref.current as Node, {
      attributes: true,
    });
  }, []);

  return (
    <Text<L>
      ref={ref}
      className={CSS.BM("text", "editable")}
      onBlur={() => setEditable(false)}
      onKeyDown={handleKeyDown}
      onKeyUp={(e) => {
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
    // @ts-expect-error
    return <Text<L> {...props}>{value}</Text>;

  return (
    <>
      {/* @ts-expect-error */}
      <Editable<L>
        allowDoubleClick={allowDoubleClick}
        onChange={onChange}
        value={value}
        {...props}
      />
    </>
  );
};
