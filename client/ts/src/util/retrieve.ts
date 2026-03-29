// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type primitive } from "@synnaxlabs/x";

import { MultipleFoundError, NotFoundError } from "@/errors";

export type SingleParamAnalysisResult<
  T extends primitive.Value,
  K extends PartialTypeNameRecord<T>,
> = T extends unknown
  ? { single: true; variant: K[keyof K]; normalized: T[]; actual: T }
  : never;

export type MultiParamAnalysisResult<
  T extends primitive.Value,
  K extends PartialTypeNameRecord<T>,
> = T extends unknown
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
        : T extends (...args: any[]) => any
          ? "function"
          : "object";

export type PartialTypeNameRecord<T extends primitive.Value> = Partial<
  Record<TypeName<T>, string>
>;

export type ParamAnalysisResult<
  T extends primitive.Value,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
> = SingleParamAnalysisResult<T, K> | MultiParamAnalysisResult<T, K>;

export interface AnalyzeParamsOptions {
  convertNumericStrings?: boolean;
}

export const analyzeParams = <
  T extends primitive.Value = primitive.Value,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
>(
  args: T extends unknown ? T | T[] : never,
  variantMap: K,
  { convertNumericStrings = true }: AnalyzeParamsOptions = {},
): ParamAnalysisResult<T, K> => {
  const isSingle = !Array.isArray(args);
  let normal = array.toArray(args);
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
