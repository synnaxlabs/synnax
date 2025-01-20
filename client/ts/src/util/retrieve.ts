// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Primitive, toArray } from "@synnaxlabs/x";

import { MultipleFoundError, NotFoundError } from "@/errors";

export type SingleParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T>,
> = T extends any
  ? { single: true; variant: K[keyof K]; normalized: T[]; actual: T }
  : never;

export type MultiParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T>,
> = T extends any
  ? { single: false; variant: K[keyof K]; normalized: T[]; actual: T[] }
  : never;

type TypeName<T> = T extends string
  ? "string"
  : T extends number
    ? "number"
    : T extends boolean
      ? "boolean"
      : T extends undefined
        ? "undefined"
        : T extends Function
          ? "function"
          : "object";

export type PartialTypeNameRecord<T extends Primitive> = Partial<
  Record<TypeName<T>, string>
>;

export type ParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
> = SingleParamAnalysisResult<T, K> | MultiParamAnalysisResult<T, K>;

export interface AnalyzeParamsOptions {
  convertNumericStrings?: boolean;
}

export const analyzeParams = <
  T extends Primitive = Primitive,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
>(
  args: T extends any ? T | T[] : never,
  variantMap: K,
  { convertNumericStrings = true }: AnalyzeParamsOptions = {},
): ParamAnalysisResult<T, K> => {
  const isSingle = !Array.isArray(args);
  let normal = toArray(args);
  const first = normal[0];
  const t = typeof first;
  let variant: K[keyof K];
  if (t === "string" && convertNumericStrings)
    if (!isNaN(parseInt(first as string)) && "number" in variantMap) {
      variant = variantMap.number as K[keyof K];
      normal = normal.map((n) => parseInt(n as string));
    } else variant = variantMap[t as TypeName<T>];
  else variant = variantMap[t as TypeName<T>];
  return {
    single: isSingle,
    variant,
    normalized: normal,
    actual: args,
  } as ParamAnalysisResult<T, K>;
};

export const checkForMultipleOrNoResults = <T, R>(
  name: string,
  params: T | T[],
  results: R[],
  isSingle: boolean,
): void => {
  if (!isSingle) return;
  if (results.length === 0)
    throw new NotFoundError(`${name} not found matching ${JSON.stringify(params)}`);
  if (results.length > 1)
    throw new MultipleFoundError(
      `Expected one ${name} matching ${JSON.stringify(params)}, but found ${results.length}`,
    );
};
