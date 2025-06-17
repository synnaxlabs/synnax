import { type FC } from "react";

import { Icon } from "@/icon";
import { Text } from "@/text";
import { Minimal } from "@/vis/slate/symbols/Base";

export const createOperator = (
  operator: string,
  single: boolean = false,
  inputIcon: Icon.IconFC = Icon.Value,
  outputIcon: Icon.IconFC = Icon.Boolean,
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
        code
        shade={10}
        style={{ transform: "scale(1.2) translateY(-2%)", padding: "1rem" }}
      >
        {operator}
      </Text.Text>
    </Minimal>
  );
  C.displayName = `Operator(${operator})`;
  return C;
};

export const Add = createOperator("+", false, Icon.Value, Icon.Value);
export const Subtract = createOperator("-", false, Icon.Value, Icon.Value);
export const Multiply = createOperator("*", false, Icon.Value, Icon.Value);
export const Divide = createOperator("/", false, Icon.Value, Icon.Value);
export const GreaterThan = createOperator(">", false, Icon.Value, Icon.Boolean);
export const LessThan = createOperator("<", false, Icon.Value, Icon.Boolean);
export const Equal = createOperator("=", false, Icon.Value, Icon.Boolean);
export const NotEqual = createOperator("≠", false, Icon.Value, Icon.Boolean);
export const GreaterThanOrEqual = createOperator("≥", false, Icon.Value, Icon.Boolean);
export const LessThanOrEqual = createOperator("≤", false, Icon.Value, Icon.Boolean);
export const And = createOperator("&&", false, Icon.Boolean, Icon.Boolean);
export const Or = createOperator("||", false, Icon.Boolean, Icon.Boolean);
export const Not = createOperator("!", true, Icon.Boolean, Icon.Boolean);
