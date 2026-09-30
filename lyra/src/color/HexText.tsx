// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/HexText.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/css";
import { Input } from "@/input";

const HEX_DIGITS = /^#?[0-9a-f]+$/i;
const HEX_WITHOUT_ALPHA = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i;
const COMPLETE_HEX = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;

/**
 * Parses what the user typed or pasted into a hex box. Bare hex digits need no `#`,
 * and any other CSS color also parses. A hex with no alpha digits keeps `alpha`, or is
 * opaque when `alpha` is 0, so a new color never lands invisible.
 */
export const parseHexInput = (text: string, alpha: number): color.Color | undefined => {
  const trimmed = text.trim();
  const isHex = HEX_DIGITS.test(trimmed);
  const parsed = color.fromCSS(
    isHex && !trimmed.startsWith("#") ? `#${trimmed}` : trimmed,
  );
  if (parsed == null) return undefined;
  if (!HEX_WITHOUT_ALPHA.test(trimmed)) return parsed;
  return color.setAlpha(parsed, alpha === 0 ? 1 : alpha);
};

export interface HexTextProps extends Omit<
  Input.TextProps,
  "value" | "onChange" | "onBlur" | "onKeyDown" | "onPaste"
> {
  /** The hex digits to show while the user is not editing. */
  text: string;
  /** The alpha a hex without alpha digits keeps. */
  alpha: number;
  onChange: (value: color.Color) => void;
  /** Called when the user commits an empty box. Without it, an empty box reverts. */
  onClear?: () => void;
}

/**
 * A text box for a hex color. A complete 6 or 8 digit hex applies as the user types,
 * a paste applies at once, and anything else applies on Enter or blur. With
 * `onlyChangeOnBlur`, everything applies on Enter or blur.
 */
export const HexText = ({
  text,
  alpha,
  onChange,
  onClear,
  onlyChangeOnBlur = false,
  className,
  ...rest
}: HexTextProps): ReactElement => {
  const [draft, setDraft] = useState<string | null>(null);
  const commit = (raw: string): void => {
    setDraft(null);
    if (raw.trim() === "" && onClear != null) return onClear();
    const parsed = parseHexInput(raw, alpha);
    if (parsed != null) onChange(parsed);
  };
  return (
    <Input.Text
      className={CSS.cls(CSS.B("color-hex-text"), className)}
      value={draft ?? text}
      onChange={(raw) => {
        setDraft(raw);
        // "fff" is also the start of "ffffff", so shorter forms wait for a commit.
        if (onlyChangeOnBlur || !COMPLETE_HEX.test(raw.trim())) return;
        const parsed = parseHexInput(raw, alpha);
        if (parsed != null) onChange(parsed);
      }}
      onBlur={() => {
        if (draft != null) commit(draft);
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter" && draft != null) commit(draft);
      }}
      onPaste={(e) => {
        const parsed = parseHexInput(e.clipboardData.getData("text"), alpha);
        if (parsed == null) return;
        e.preventDefault();
        if (onlyChangeOnBlur) return setDraft(color.hex(parsed).slice(1));
        setDraft(null);
        onChange(parsed);
      }}
      selectOnFocus
      {...rest}
    />
  );
};
