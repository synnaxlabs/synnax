// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, type primitive } from "@synnaxlabs/x";
import { type z } from "zod";

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
    if (!isNaN(parseInt(first as string, 10)) && "number" in variantMap) {
      variant = variantMap.number as K[keyof K];
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

const isSingleShape = (
  args: readonly filter.FilterArg[],
  identifyingFields: ReadonlySet<string>,
): boolean => {
  if (args.length !== 1) return false;
  const arg = args[0];
  if (typeof arg !== "object" || arg === null || filter.isNode(arg)) return false;
  let id: string | undefined;
  for (const k of Object.keys(arg)) {
    if (filter.BASE_OPTION_KEYS.has(k)) continue;
    if (!identifyingFields.has(k) || id !== undefined) return false;
    id = k;
  }
  if (id === undefined) return false;
  const v = (arg as Record<string, unknown>)[id];
  return typeof v === "string" || typeof v === "number" || typeof v === "bigint";
};

const extractOptions = (
  arg: object,
  into: BaseRetrieveRequest,
): void => {
  const source = arg as Record<string, unknown>;
  for (const k of filter.BASE_OPTION_KEYS) {
    const v = source[k];
    if (v === undefined) continue;
    (into as Record<string, unknown>)[k] = v;
  }
};

/**
 * BaseRetrieveRequest is the canonical wire shape every generated retrieveReqZ
 * conforms to. The where slot carries the composable filter tree; the rest are
 * shared retrieval options (subset of {@link filter.BASE_OPTION_KEYS}). Per-
 * entity schemas may omit fields they don't support (e.g. includeStatus), but
 * never add ones outside this shape.
 */
export interface BaseRetrieveRequest {
  where?: object;
  searchTerm?: string;
  limit?: number;
  offset?: number;
  orderBy?: Record<string, unknown>;
  includeStatus?: boolean;
}

/**
 * RetrieveDescriptor names the wire path, response item key, filter descriptor,
 * identifying-field set, and Zod schemas for an entity's retrieve endpoint. It
 * is intended to be assembled from generated metadata (descriptor, schemas,
 * itemsKey, identifyingFields) and supplied to {@link executeRetrieve}.
 */
export interface RetrieveDescriptor<Payload, ItemsKey extends string> {
  path: string;
  entityName: string;
  reqZ: z.ZodType<BaseRetrieveRequest>;
  resZ: z.ZodType<{ [K in ItemsKey]?: Payload[] | null }>;
  itemsKey: ItemsKey;
  filter: filter.Descriptor;
  identifyingFields: ReadonlySet<string>;
}

/**
 * executeRetrieve is the shared driver for every entity's retrieve method. It
 * detects single-vs-multi from the args' shape using the generated identifying
 * fields, extracts option keys (limit/offset/searchTerm/orderBy/includeStatus),
 * converts filter args to the wire shape, dispatches the request, and maps
 * payloads through sugar. When invoked in single shape, asserts exactly-one
 * result via {@link NotFoundError}/{@link MultipleFoundError}.
 */
export const executeRetrieve = async <Payload, Entity, ItemsKey extends string>(
  desc: RetrieveDescriptor<Payload, ItemsKey>,
  client: UnaryClient,
  args: readonly filter.FilterArg[],
  sugar: (p: Payload) => Entity,
): Promise<Entity | Entity[]> => {
  const single = isSingleShape(args, desc.identifyingFields);
  const filterArgs: filter.FilterArg[] = [];
  const options: Record<string, unknown> = {};
  for (const arg of args) {
    if (filter.isNode(arg)) {
      filterArgs.push(arg);
      continue;
    }
    if (typeof arg === "object" && arg !== null) {
      extractOptions(arg, options);
      filterArgs.push(arg);
    }
  }
  const where = filter.toWire(filterArgs, desc.filter);
  const res = await sendRequired(
    client,
    desc.path,
    { ...options, where },
    desc.reqZ,
    desc.resZ,
  );
  const rawItems = (res as Record<string, unknown>)[desc.itemsKey] as
    | Payload[]
    | null
    | undefined;
  const sugared = (rawItems ?? []).map(sugar);
  if (single) {
    checkForMultipleOrNoResults(desc.entityName, args[0], sugared, true);
    return sugared[0];
  }
  return sugared;
};
