// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { evaluate, Unit } from "mathjs";
import { type ReactElement, useCallback, useEffect } from "react";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { DragButton, type DragButtonExtraProps } from "@/input/DragButton";
import { Text, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";
import { Triggers } from "@/triggers";

/** Props for {@link Numeric}. */
export interface NumericProps
  extends
    Omit<TextProps, "type" | "onBlur" | "value" | "onChange">,
    DragButtonExtraProps,
    Control<number> {
  /** Whether focusing selects the whole value. Defaults to true. */
  selectOnFocus?: boolean;
  /** Whether to show the drag handle that scrubs the value. Defaults to true. */
  showDragHandle?: boolean;
  /** Clamps the committed value. */
  bounds?: bounds.Crude;
  onBlur?: () => void;
  /** Unit suffix shown after the value, e.g. "Hz". */
  units?: string;
  /// When set, a value equal to emptyValue renders as an empty input (showing the
  /// placeholder) and clearing the input on blur emits emptyValue via onChange. Useful
  /// for representing a sentinel "unset"/"auto" state without showing the raw number.
  emptyValue?: number;
}

/**
 * A number input. It accepts any math expression `mathjs` can evaluate, so a user can
 * type `2 * 60` or `1 kHz`, and it commits on blur or Enter rather than per keystroke.
 * A drag handle scrubs the value.
 *
 * @example <Input.Numeric value={rate} onChange={setRate} units="Hz" />
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
  units,
  size,
  color,
  emptyValue,
  ...rest
}: NumericProps): ReactElement => {
  const isEmpty = emptyValue != null && value === emptyValue;
  // We need to keep the actual value as a valid number, but we need to let the user
  // input an invalid value that may eventually be valid, so we need to keep the
  // internal value as a string in state.
  const [internalValue, setInternalValue, internalValueRef] = useCombinedStateAndRef(
    isEmpty ? "" : value.toString(),
  );
  const [isValueValid, setIsValueValid, isValueValidRef] =
    useCombinedStateAndRef<boolean>(true);
  const valueRef = useSyncedRef(value);

  const updateActualValue = useCallback(() => {
    // This just means we never actually modified the input
    if (isValueValidRef.current) return;
    setIsValueValid(true);
    const raw = internalValueRef.current.trim();
    if (raw === "" && emptyValue != null) {
      onChange?.(emptyValue);
      return;
    }
    let v = null;
    try {
      const ev = evaluate(internalValueRef.current);
      // Sometimes mathjs returns a Unit object, so we need to convert it to a number.
      if (ev instanceof Unit) v = ev.toNumber();
      else if (typeof ev === "number" && !isNaN(ev)) v = ev;
    } catch {
      v = null;
    }
    if (v != null) onChange?.(bounds.clamp(propsBounds, v));
    else
      setInternalValue(
        emptyValue != null && valueRef.current === emptyValue
          ? ""
          : valueRef.current.toString(),
      );
  }, [onChange, setInternalValue, emptyValue]);

  const updateActualValueRef = useSyncedRef(updateActualValue);

  const handleBlur = useCallback(() => {
    onBlur?.();
    updateActualValue();
  }, [onBlur, updateActualValue]);

  // Sometimes we don't blur the component before it unmounts, so this makes
  // sure we try to update the actual value on unmount.
  useEffect(() => () => updateActualValueRef.current?.(), []);

  const handleChange = useCallback(
    (v: string) => {
      setIsValueValid(false);
      setInternalValue(v);
    },
    [setInternalValue, setIsValueValid],
  );

  // If the value is valid, use the actual value, otherwise use the internal value.
  // When the value matches emptyValue, render as an empty string so the placeholder
  // shows through.
  const value_ = isValueValid ? (isEmpty ? "" : value.toString()) : internalValue;

  const onDragChange = useCallback(
    (value: number) => {
      const next = bounds.clamp(propsBounds, Math.round(value));
      // A gated input parks the drag in the internal value, so the text tracks the
      // pointer and the release commits through the same blur path typing uses.
      if (onlyChangeOnBlur) {
        setIsValueValid(false);
        setInternalValue(next.toString());
        return;
      }
      setIsValueValid(true);
      onChange?.(next);
    },
    [onChange, onlyChangeOnBlur, setInternalValue, setIsValueValid],
  );

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
      className={className}
      value={value_}
      onChange={handleChange}
      disabled={disabled}
      selectOnFocus={selectOnFocus}
      // When the user hits 'Enter', we should try to evaluate the input and update the
      // actual value.
      onKeyDown={(e) => {
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
          value={value}
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
