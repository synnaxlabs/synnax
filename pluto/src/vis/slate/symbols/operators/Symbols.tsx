import { type FC } from "react";

import { Align } from "@/align";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const createComparison = (operator: string): FC => {
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
      <Handle.Sink location="left" id="y" style={{ top: "66%" }} />
      <Handle.Source location="right" id="value" style={{ top: "50%" }} />
    </Align.Space>
  );
  C.displayName = `Comparison(${operator})`;
  return C;
};

export const GreaterThan = createComparison(">");
export const LessThan = createComparison("<");
export const Equal = createComparison("=");
export const NotEqual = createComparison("≠");
export const GreaterThanOrEqual = createComparison("≥");
export const LessThanOrEqual = createComparison("≤");
export const Add = createComparison("+");
export const Subtract = createComparison("-");
export const Multiply = createComparison("*");
export const Divide = createComparison("/");
