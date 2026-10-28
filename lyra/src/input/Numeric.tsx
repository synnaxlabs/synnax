// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { DragButton, type DragButtonExtraProps } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";
import { Triggers } from "@/triggers";

const parseNumber = (raw: string): number | null => {
  const n = Number(raw);
  return raw.trim() !== "" && Number.isFinite(n) ? n : null;
};

// mathjs is heavy and only an expression or a unit needs it, so it loads on demand.
const evaluateExpression = async (raw: string): Promise<number | null> => {
  const { evaluate, Unit } = await import("mathjs");
  try {
    const ev = evaluate(raw);
    if (ev instanceof Unit) return ev.toNumber();
    if (typeof ev === "number" && !isNaN(ev)) return ev;
  } catch {
    return null;
  }
  return null;
};

/** Props for {@link Numeric}. */
export interface NumericProps
  extends
    Omit<TextProps, "type" | "onBlur" | "value" | "onChange">,
    DragButtonExtraProps,
    Control<number | undefined, number> {
  /** Whether focusing selects the whole value. Defaults to true. */
  selectOnFocus?: boolean;
  /** Whether to show the drag handle that scrubs the value. Defaults to true. */
  showDragHandle?: boolean;
  /** Clamps the committed value. */
  bounds?: bounds.Crude;
  onBlur?: () => void;
  /**
   * A stored sentinel the schema gives a meaning such as "auto". A value equal to it
   * renders empty, and clearing the input commits it. Equality is `Object.is`, so NaN
   * works as the sentinel.
   */
  emptyValue?: number;
}

/**
 * A number input. It accepts any math expression `mathjs` can evaluate, so a user can
 * type `2 * 60` or `1 kHz`, and it commits on blur or Enter rather than per keystroke.
 * A drag handle scrubs the value.
 *
 * @example <Input.Numeric value={rate} onChange={setRate} endContent="Hz" />
 */
export const Numeric = ({
  ref,
  onChange,
  value,
  dragDirection,
  showDragHandle = true,
  dragScale,
  selectOnFocus = true,
  bounds: propsBounds = bounds.INFINITE,
  onlyChangeOnBlur = false,
  resetValue,
  variant = "outlined",
  preview,
  className,
  children,
  disabled,
  onBlur,
  size,
  color,
  emptyValue,
  ...rest
}: NumericProps): ReactElement => {
  // An absent value renders empty, so the placeholder reads as the unset state.
  const isEmpty = value == null || Object.is(value, emptyValue);
  // We need to keep the actual value as a valid number, but we need to let the user
  // input an invalid value that may eventually be valid, so we need to keep the
  // internal value as a string in state.
  const [internalValue, setInternalValue, internalValueRef] = useCombinedStateAndRef(
    isEmpty ? "" : value.toString(),
  );
  const [isValueValid, setIsValueValid, isValueValidRef] =
    useCombinedStateAndRef<boolean>(true);
  const valueRef = useSyncedRef(value);
  const boundsRef = useSyncedRef(propsBounds);
  // A later commit or an Esc supersedes an expression that is still evaluating.
  const evaluation = useRef(0);

  const updateActualValue = () => {
    // This just means we never actually modified the input
    if (isValueValidRef.current) return;
    setIsValueValid(true);
    const id = ++evaluation.current;
    const raw = internalValueRef.current.trim();
    if (raw === "" && emptyValue != null) {
      onChange?.(emptyValue);
      return;
    }
    if (raw === "" && valueRef.current == null) return;
    const commit = (v: number | null): void => {
      if (v != null) onChange?.(bounds.clamp(boundsRef.current, v));
      else
        setInternalValue(
          valueRef.current == null || Object.is(valueRef.current, emptyValue)
            ? ""
            : valueRef.current.toString(),
        );
    };
    const plain = parseNumber(raw);
    if (plain != null) return commit(plain);
    void evaluateExpression(raw).then((v) => {
      if (id === evaluation.current) commit(v);
    });
  };

  const updateActualValueRef = useSyncedRef(updateActualValue);

  const handleBlur = () => {
    onBlur?.();
    updateActualValue();
  };

  // Sometimes we don't blur the component before it unmounts, so this makes
  // sure we try to update the actual value on unmount.
  useEffect(() => () => updateActualValueRef.current?.(), []);

  const handleChange = (v: string) => {
    setIsValueValid(false);
    setInternalValue(v);
  };

  // If the value is valid, use the actual value, otherwise use the internal value.
  const value_ = isValueValid ? (isEmpty ? "" : value.toString()) : internalValue;

  const onDragChange = (value: number) => {
    const next = bounds.clamp(boundsRef.current, Math.round(value));
    // A gated input parks the drag in the internal value, so the text tracks the
    // pointer and the release commits through the same blur path typing uses.
    if (onlyChangeOnBlur) {
      setIsValueValid(false);
      setInternalValue(next.toString());
      return;
    }
    setIsValueValid(true);
    onChange?.(next);
  };

  if (dragScale == null && bounds.isFinite(propsBounds))
    dragScale = {
      x: bounds.span(propsBounds) * 0.01,
      y: bounds.span(propsBounds) * 0.02,
    };

  if (preview === true) showDragHandle = false;

  return (
    <Text
      ref={ref}
      type="text"
      variant={variant}
      preview={preview}
      className={CSS.cls(CSS.BM("input", "numeric"), className)}
      value={value_}
      onChange={handleChange}
      disabled={disabled}
      selectOnFocus={selectOnFocus}
      // When the user hits 'Enter', we should try to evaluate the input and update the
      // actual value.
      onKeyDown={(e) => {
        // Esc cancels typed text, so the blur that follows commits nothing.
        if (e.key === "Escape") {
          evaluation.current++;
          return setIsValueValid(true);
        }
        if (Triggers.eventKey(e) !== "Enter") return;
        updateActualValue();
        onBlur?.();
      }}
      onBlur={handleBlur}
      size={size}
      color={color}
      {...rest}
    >
      {showDragHandle && (
        <DragButton
          direction={dragDirection}
          value={value ?? 0}
          onChange={onDragChange}
          dragScale={dragScale}
          resetValue={resetValue}
          onBlur={handleBlur}
          size={size}
          color={color}
          disabled={disabled}
        />
      )}
      {children}
    </Text>
  );
};
