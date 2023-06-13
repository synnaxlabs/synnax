// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useLayoutEffect, useRef, useState } from "react";
import type { KeyboardEvent, ReactElement } from "react";

import { CSS } from "@/core/css";
import { InputControl } from "@/core/std/Input";
import { Text, TextProps } from "@/core/std/Typography/Text";
import { TypographyLevel } from "@/core/std/Typography/types";

import "@/core/std/Typography/TextEditable.css";

export type TextEditableProps<L extends TypographyLevel = "h1"> = Omit<
  TextProps<L>,
  "children" | "onChange"
> &
  InputControl<string>;

const NOMINAL_EXIT_KEYS = ["Escape", "Enter"];

export const TextEditable = <L extends TypographyLevel = "h1">({
  onChange,
  value,
  ...props
}: TextEditableProps<L>): ReactElement => {
  const [editable, setEditable] = useState(false);
  const ref = useRef<HTMLElement>(null);

  const handleDoubleClick = (): void => setEditable(true);

  const handleKeyDown = (e: KeyboardEvent<HTMLParagraphElement>): void => {
    if (!editable || !NOMINAL_EXIT_KEYS.includes(e.key) || ref.current == null) return;
    const el = ref.current;
    setEditable(false);
    onChange?.(el.innerText);
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

  return (
    // @ts-expect-error
    <Text<L>
      ref={ref}
      className={CSS.BM("text", "editable")}
      onBlur={() => setEditable(false)}
      onKeyDown={handleKeyDown}
      onDoubleClick={handleDoubleClick}
      contentEditable={editable}
      suppressContentEditableWarning
      {...props}
    >
      {value}
    </Text>
  );
};
