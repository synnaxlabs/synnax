// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/TriggerClear.css";

import { primitive, type record } from "@synnaxlabs/x";
import type { KeyboardEvent, ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useClear, useSelected } from "@/select/Context";

/** Props for {@link useTriggerClear}. */
export interface UseTriggerClearProps {
  /** Whether the trigger may clear its selection. */
  clearable: boolean;
  disabled?: boolean;
  preview?: boolean;
}

/** The clear action that {@link useTriggerClear} returns. */
export interface UseTriggerClearReturn {
  /** Whether the selection can be cleared now. A zero key means none. */
  canClear: boolean;
  /** The class of a trigger that shows a {@link TriggerClear}. */
  className: string;
  /** Clears the selection on Delete or Backspace. */
  handleKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
}

/**
 * The clear action of the trigger of a single selection. A key with no entry, such as a
 * deleted resource, can still be cleared.
 */
export const useTriggerClear = <K extends record.Key>({
  clearable,
  disabled,
  preview,
}: UseTriggerClearProps): UseTriggerClearReturn => {
  const [selected] = useSelected<K>();
  const clear = useClear();
  const canClear =
    clearable && primitive.isNonZero(selected) && disabled !== true && preview !== true;
  // The clear button is not a tab stop, so the focused trigger takes the key instead.
  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if (!canClear || (e.key !== "Delete" && e.key !== "Backspace")) return;
    e.preventDefault();
    clear();
  };
  return {
    canClear,
    className: CSS.cls(CSS.M("reveals"), CSS.B("select-clearable")),
    handleKeyDown,
  };
};

/** Props for {@link TriggerClear}. */
export interface TriggerClearProps {
  /** Names the selection in the label of the clear button. */
  name?: string;
  /** The glyph that the clear button covers. */
  children: ReactElement;
}

/**
 * Covers a glyph of a trigger with a button that clears the selection. The button shows
 * while a trigger with the class from {@link useTriggerClear} is hovered.
 */
export const TriggerClear = ({ name, children }: TriggerClearProps): ReactElement => {
  const clear = useClear();
  return (
    <span className={CSS.B("select-clear")}>
      {children}
      <Button.Close
        el="div"
        role="button"
        aria-label={`Clear ${name ?? "selection"}`}
        className={CSS.BE("select-clear", "button")}
        onClick={clear}
      />
    </span>
  );
};
