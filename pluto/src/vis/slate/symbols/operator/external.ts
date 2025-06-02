import { Add, Divide, Multiply, Subtract } from "@/vis/slate/symbols/operator/Symbols";

const OPERATOR_ADD = {
  key: "operator.add",
  name: "Add",
  Form: () => null,
  Symbol: Add,
  Preview: Add,
  defaultProps: () => ({}),
  zIndex: 0,
};

const OPERATOR_SUBTRACT = {
  key: "operator.subtract",
  name: "Subtract",
  Form: () => null,
  Symbol: Subtract,
  Preview: Subtract,
  defaultProps: () => ({}),
  zIndex: 0,
};

const OPERATOR_MULTIPLY = {
  key: "operator.multiply",
  name: "Multiply",
  Form: () => null,
  Symbol: Multiply,
  Preview: Multiply,
  defaultProps: () => ({}),
  zIndex: 0,
};

const OPERATOR_DIVIDE = {
  key: "operator.divide",
  name: "Divide",
  Form: () => null,
  Symbol: Divide,
  Preview: Divide,
  defaultProps: () => ({}),
  zIndex: 0,
};

export const REGISTRY = {
  [OPERATOR_ADD.key]: OPERATOR_ADD,
  [OPERATOR_SUBTRACT.key]: OPERATOR_SUBTRACT,
  [OPERATOR_MULTIPLY.key]: OPERATOR_MULTIPLY,
  [OPERATOR_DIVIDE.key]: OPERATOR_DIVIDE,
};
