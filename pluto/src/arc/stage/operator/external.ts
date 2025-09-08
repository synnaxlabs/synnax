import {
  Add,
  And,
  Divide,
  Equal,
  GreaterThan,
  GreaterThanOrEqual,
  LessThan,
  LessThanOrEqual,
  Multiply,
  Not,
  NotEqual,
  Or,
  Subtract,
} from "@/arc/stage/operator/Symbols";

const ADD = {
  key: "operator.add",
  name: "Add",
  group: "operator",
  Form: () => null,
  Symbol: Add,
  Preview: Add,
  defaultProps: () => ({}),
  zIndex: 0,
};

const SUBTRACT = {
  key: "operator.subtract",
  name: "Subtract",
  group: "operator",
  Form: () => null,
  Symbol: Subtract,
  Preview: Subtract,
  defaultProps: () => ({}),
  zIndex: 0,
};

const MULTIPLY = {
  key: "operator.multiply",
  name: "Multiply",
  Form: () => null,
  Symbol: Multiply,
  Preview: Multiply,
  defaultProps: () => ({}),
  zIndex: 0,
};

const DIVIDE = {
  key: "operator.divide",
  name: "Divide",
  Form: () => null,
  Symbol: Divide,
  Preview: Divide,
  defaultProps: () => ({}),
  zIndex: 0,
};

const GREATER_THAN = {
  key: "operator.gt",
  name: "Greater Than",
  Form: () => null,
  Symbol: GreaterThan,
  Preview: GreaterThan,
  defaultProps: () => ({}),
  zIndex: 0,
};

const LESS_THAN = {
  key: "operator.lt",
  name: "Less Than",
  Form: () => null,
  Symbol: LessThan,
  Preview: LessThan,
  defaultProps: () => ({}),
  zIndex: 0,
};

const EQUAL = {
  key: "operator.eq",
  name: "Equal",
  Form: () => null,
  Symbol: Equal,
  Preview: Equal,
  defaultProps: () => ({}),
  zIndex: 0,
};

const NOT_EQUAL = {
  key: "operator.ne",
  name: "Not Equal",
  Form: () => null,
  Symbol: NotEqual,
  Preview: NotEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

const GREATER_THAN_OR_EQUAL = {
  key: "operator.ge",
  name: "Greater Than or Equal",
  Form: () => null,
  Symbol: GreaterThanOrEqual,
  Preview: GreaterThanOrEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

const LESS_THAN_OR_EQUAL = {
  key: "operator.le",
  name: "Less Than or Equal",
  Form: () => null,
  Symbol: LessThanOrEqual,
  Preview: LessThanOrEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

const AND = {
  key: "operator.and",
  name: "And",
  Form: () => null,
  Symbol: And,
  Preview: And,
  defaultProps: () => ({}),
  zIndex: 0,
};

const OR = {
  key: "operator.or",
  name: "Or",
  Form: () => null,
  Symbol: Or,
  Preview: Or,
  defaultProps: () => ({}),
  zIndex: 0,
};

const NOT = {
  key: "operator.not",
  name: "Not",
  Form: () => null,
  Symbol: Not,
  Preview: Not,
  defaultProps: () => ({}),
  zIndex: 0,
};

export const SYMBOLS = {
  [ADD.key]: ADD,
  [SUBTRACT.key]: SUBTRACT,
  [MULTIPLY.key]: MULTIPLY,
  [DIVIDE.key]: DIVIDE,
  [GREATER_THAN.key]: GREATER_THAN,
  [LESS_THAN.key]: LESS_THAN,
  [EQUAL.key]: EQUAL,
  [NOT_EQUAL.key]: NOT_EQUAL,
  [GREATER_THAN_OR_EQUAL.key]: GREATER_THAN_OR_EQUAL,
  [LESS_THAN_OR_EQUAL.key]: LESS_THAN_OR_EQUAL,
  [AND.key]: AND,
  [OR.key]: OR,
  [NOT.key]: NOT,
};
