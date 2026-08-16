// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod/v4";

import { createOperator } from "@/arc/graph/node/common/create";
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
} from "@/arc/graph/node/operator/Operator";

const add = createOperator({ key: "add", name: "Add", Symbol: Add });
const subtract = createOperator({
  key: "subtract",
  name: "Subtract",
  Symbol: Subtract,
});
const multiply = createOperator({
  key: "multiply",
  name: "Multiply",
  Symbol: Multiply,
});
const divide = createOperator({ key: "divide", name: "Divide", Symbol: Divide });
const greaterThan = createOperator({
  key: "gt",
  name: "Greater Than",
  Symbol: GreaterThan,
});
const lessThan = createOperator({ key: "lt", name: "Less Than", Symbol: LessThan });
const equal = createOperator({ key: "eq", name: "Equal", Symbol: Equal });
const notEqual = createOperator({ key: "ne", name: "Not Equal", Symbol: NotEqual });
const greaterThanOrEqual = createOperator({
  key: "ge",
  name: "Greater Than or Equal",
  Symbol: GreaterThanOrEqual,
});
const lessThanOrEqual = createOperator({
  key: "le",
  name: "Less Than or Equal",
  Symbol: LessThanOrEqual,
});
const and = createOperator({ key: "and", name: "And", Symbol: And });
const or = createOperator({ key: "or", name: "Or", Symbol: Or });
const not = createOperator({ key: "not", name: "Not", Symbol: Not });

export const REGISTRY = {
  add: add.spec,
  subtract: subtract.spec,
  multiply: multiply.spec,
  divide: divide.spec,
  gt: greaterThan.spec,
  lt: lessThan.spec,
  eq: equal.spec,
  ne: notEqual.spec,
  ge: greaterThanOrEqual.spec,
  le: lessThanOrEqual.spec,
  and: and.spec,
  or: or.spec,
  not: not.spec,
};

export const configZ = z.discriminatedUnion("type", [
  add.configZ,
  subtract.configZ,
  multiply.configZ,
  divide.configZ,
  greaterThan.configZ,
  lessThan.configZ,
  equal.configZ,
  notEqual.configZ,
  greaterThanOrEqual.configZ,
  lessThanOrEqual.configZ,
  and.configZ,
  or.configZ,
  not.configZ,
]);
