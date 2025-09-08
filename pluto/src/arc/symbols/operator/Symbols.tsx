// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";

import { Minimal } from "@/arc/symbols/Base";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const createOperator = (
  operator: string,
  single: boolean = false,
  inputIcon: Icon.FC = Icon.Number,
  outputIcon: Icon.FC = Icon.Boolean,
): FC => {
  const C = () => (
    <Minimal
      sinks={[
        { key: "x", Icon: inputIcon },
        ...(single ? [] : [{ key: "y", Icon: inputIcon }]),
      ]}
      sources={[{ key: "value", Icon: outputIcon }]}
      centerSources
      centerSinks={single}
    >
      <Text.Text
        level="h4"
        weight={500}
        variant="code"
        color={10}
        style={{ transform: "scale(1.2) translateY(-2%)", padding: "1rem" }}
      >
        {operator}
      </Text.Text>
    </Minimal>
  );
  C.displayName = `Operator(${operator})`;
  return C;
};

export const Add = createOperator("+", false, Icon.Number, Icon.Number);
export const Subtract = createOperator("-", false, Icon.Number, Icon.Number);
export const Multiply = createOperator("*", false, Icon.Number, Icon.Number);
export const Divide = createOperator("/", false, Icon.Number, Icon.Number);
export const GreaterThan = createOperator(">", false, Icon.Number, Icon.Boolean);
export const LessThan = createOperator("<", false, Icon.Number, Icon.Boolean);
export const Equal = createOperator("=", false, Icon.Number, Icon.Boolean);
export const NotEqual = createOperator("≠", false, Icon.Number, Icon.Boolean);
export const GreaterThanOrEqual = createOperator("≥", false, Icon.Number, Icon.Boolean);
export const LessThanOrEqual = createOperator("≤", false, Icon.Number, Icon.Boolean);
export const And = createOperator("&&", false, Icon.Boolean, Icon.Boolean);
export const Or = createOperator("||", false, Icon.Boolean, Icon.Boolean);
export const Not = createOperator("!", true, Icon.Boolean, Icon.Boolean);
