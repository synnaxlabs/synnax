// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Control, Value, OptionalControl } from "./types";

export const usePassthrough = <I extends Value = Value, O extends Value = I>({
  value,
  onChange,
  initialValue,
}: OptionalControl<I, O> & { initialValue: I }): [
  Control<I, O>["value"],
  Control<I, O>["onChange"]
] => {
  const [internalValue, setInternalValue] = useState<I>(value ?? initialValue);
  if (value != null && onChange != null) return [value, onChange];
  return [internalValue, setInternalValue as unknown as Control<I, O>["onChange"]];
};
