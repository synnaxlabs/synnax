// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { InputControl, InputValue, OptionalInputControl } from "./types";

export const usePassThroughInputControl = <
  I extends InputValue = InputValue,
  O extends InputValue = I
>({
  value,
  onChange,
  initialValue,
}: OptionalInputControl<I, O> & { initialValue: I }): [
  InputControl<I, O>["value"],
  InputControl<I, O>["onChange"]
] => {
  const [internalValue, setInternalValue] = useState<I>(value ?? initialValue);
  if (value != null && onChange != null) return [value, onChange];
  return [internalValue, setInternalValue as unknown as InputControl<I, O>["onChange"]];
};
