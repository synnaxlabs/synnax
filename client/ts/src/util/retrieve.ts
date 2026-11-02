// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, filter, type primitive } from "@synnaxlabs/x";

import { MultipleFoundError, NotFoundError } from "@/errors";

export type SingleParamAnalysisResult<
  T extends primitive.Value,
  K extends PartialTypeNameRecord<T>,
> = T extends unknown
  ? { single: true; variant: K[TypeName<T>]; normalized: T[]; actual: T }
  : never;

export type MultiParamAnalysisResult<
  T extends primitive.Value,
  K extends PartialTypeNameRecord<T>,
> = T extends unknown
  ? { single: false; variant: K[TypeName<T>]; normalized: T[]; actual: T[] }
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
  let variant: K[TypeName<T>];
  if (t === "string" && convertNumericStrings)
    if (!isNaN(parseInt(first as string, 10)) && "number" in variantMap) {
      variant = variantMap.number as K[TypeName<T>];
      normal = normal.map((n) => parseInt(n as string, 10));
    } else variant = variantMap[t as TypeName<T>];
  else variant = variantMap[t as TypeName<T>];
  return {
    single: isSingle,
    variant,
    normalized: normal,
    actual: args,
  } as ParamAnalysisResult<T, K>;
};

/**
 * Checks that a retrieve for one record found exactly one.
 * @param name - The name of the record type, such as "range".
 * @param where - The filter that selected the results, for the error message.
 * @param results - The records the retrieve found.
 * @param isSingle - Whether the retrieve was for one record. If false, nothing is
 * checked.
 * @throws {NotFoundError} if results is empty.
 * @throws {MultipleFoundError} if results holds more than one record.
 */
export const checkForMultipleOrNoResults = <R>(
  name: string,
  where: object | undefined,
  results: R[],
  isSingle: boolean,
): void => {
  if (!isSingle || results.length === 1) return;
  const description = where == null ? "" : filter.describe(where);
  const clause = description === "" ? "" : ` where ${description}`;
  if (results.length === 0) throw new NotFoundError(`Found no ${name}${clause}`);
  throw new MultipleFoundError(
    `Expected one ${name}${clause}, but found ${results.length}`,
  );
};
