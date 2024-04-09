// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { toArray, type Primitive } from "@synnaxlabs/x";

export type SingleParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T>,
> = T extends any
  ? {
      single: true;
      variant: K[keyof K];
      normalized: T[];
      actual: T;
    }
  : never;

export type MultiParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T>,
> = T extends any
  ? {
      single: false;
      variant: K[keyof K];
      normalized: T[];
      actual: T[];
    }
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

type PartialTypeNameRecord<T extends Primitive> = Partial<Record<TypeName<T>, string>>;

export type ParamAnalysisResult<
  T extends Primitive,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
> = SingleParamAnalysisResult<T, K> | MultiParamAnalysisResult<T, K>;

export const analyzeParams = <
  T extends Primitive,
  K extends PartialTypeNameRecord<T> = PartialTypeNameRecord<T>,
>(
  args: string | number | string[] | number[],
  variantMap: K,
): ParamAnalysisResult<T, K> => {
  const isSingle = !Array.isArray(args);
  const normal = toArray(args);
  const first = normal[0];
  const variant = variantMap[typeof first as TypeName<T>];
  return {
    single: isSingle,
    variant,
    normalized: normal,
    actual: args,
  } as unknown as ParamAnalysisResult<T, K>;
};
