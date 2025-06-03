import { type FC } from "react";

import { Align } from "@/align";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const createOperator = (operator: string, single: boolean = false): FC => {
  const C = () => (
    <Align.Space
      x
      style={{
        width: "5rem",
        height: "5rem",
      }}
      bordered
      background={1}
      borderShade={5}
      rounded={1}
      align="center"
      justify="center"
    >
      <Text.Text level="h4" weight={500} code>
        {operator}
      </Text.Text>
      <Handle.Sink location="left" id="x" style={{ top: "33%" }} />
      {!single && <Handle.Sink location="left" id="y" style={{ top: "66%" }} />}
      <Handle.Source location="right" id="value" style={{ top: "50%" }} />
    </Align.Space>
  );
  C.displayName = `Operator(${operator})`;
  return C;
};

export const Add = createOperator("+");
export const Subtract = createOperator("-");
export const Multiply = createOperator("*");
export const Divide = createOperator("/");
export const GreaterThan = createOperator(">");
export const LessThan = createOperator("<");
export const Equal = createOperator("=");
export const NotEqual = createOperator("≠");
export const GreaterThanOrEqual = createOperator("≥");
export const LessThanOrEqual = createOperator("≤");
export const And = createOperator("&&");
export const Or = createOperator("||");
export const Not = createOperator("!", true);
