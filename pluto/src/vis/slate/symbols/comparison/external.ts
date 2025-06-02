import {
  Equal,
  GreaterThan,
  GreaterThanOrEqual,
  LessThan,
  LessThanOrEqual,
  NotEqual,
} from "@/vis/slate/symbols/comparison/Symbols";

const COMPARISON_GT = {
  key: "comparison.gt",
  name: "Greater Than",
  Form: () => null,
  Symbol: GreaterThan,
  Preview: GreaterThan,
  defaultProps: () => ({}),
  zIndex: 0,
};

const COMPARISON_LT = {
  key: "comparison.lt",
  name: "Less Than",
  Form: () => null,
  Symbol: LessThan,
  Preview: LessThan,
  defaultProps: () => ({}),
  zIndex: 0,
};

const COMPARISON_EQ = {
  key: "comparison.eq",
  name: "Equal",
  Form: () => null,
  Symbol: Equal,
  Preview: Equal,
  defaultProps: () => ({}),
  zIndex: 0,
};

const COMPARISON_NE = {
  key: "comparison.ne",
  name: "Not Equal",
  Form: () => null,
  Symbol: NotEqual,
  Preview: NotEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

const COMPARISON_GTE = {
  key: "comparison.gte",
  name: "Greater Than or Equal",
  Form: () => null,
  Symbol: GreaterThanOrEqual,
  Preview: GreaterThanOrEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

const COMPARISON_LTE = {
  key: "comparison.lte",
  name: "Less Than or Equal",
  Form: () => null,
  Symbol: LessThanOrEqual,
  Preview: LessThanOrEqual,
  defaultProps: () => ({}),
  zIndex: 0,
};

export const REGISTRY = {
  [COMPARISON_GT.key]: COMPARISON_GT,
  [COMPARISON_LT.key]: COMPARISON_LT,
  [COMPARISON_EQ.key]: COMPARISON_EQ,
  [COMPARISON_NE.key]: COMPARISON_NE,
  [COMPARISON_GTE.key]: COMPARISON_GTE,
  [COMPARISON_LTE.key]: COMPARISON_LTE,
};
