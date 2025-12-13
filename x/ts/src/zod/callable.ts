// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/**
 * Represents a callable method schema with typed arguments and return value.
 * This is a simple object structure that can be used to define RPC method signatures.
 */
export interface Callable<
  Args extends z.ZodTypeAny = z.ZodVoid,
  Returns extends z.ZodTypeAny = z.ZodVoid,
> {
  args: Args;
  returns: Returns;
}

/**
 * Builder interface where args/returns are both the schema AND callable to chain.
 * When called as a function, returns a new builder. When accessed as a property, acts as the schema.
 */
export interface CallableBuilder<
  Args extends z.ZodTypeAny = z.ZodVoid,
  Returns extends z.ZodTypeAny = z.ZodVoid,
> {
  /** The args schema. Also callable: .args(schema) returns a new builder with that args type. */
  args: Args & (<A extends z.ZodTypeAny>(schema: A) => CallableBuilder<A, Returns>);
  /** The returns schema. Also callable: .returns(schema) returns a new builder with that return type. */
  returns: Returns & (<R extends z.ZodTypeAny>(schema: R) => CallableBuilder<Args, R>);
}

/** Symbol used to store the actual Zod schema on our proxy objects. */
const SCHEMA_SYMBOL = Symbol("zodSchema");

/**
 * Creates a proxy that acts as both a Zod schema (for property access like instanceof)
 * and a callable function (for chaining like .args(z.string())).
 */
const createSchemaProxy = <S extends z.ZodTypeAny, BuilderFn extends (...args: any[]) => any>(
  schema: S,
  builderFn: BuilderFn,
): S & BuilderFn => {
  const fn = builderFn as any;
  fn[SCHEMA_SYMBOL] = schema;

  return new Proxy(fn, {
    get(target, prop, receiver) {
      // Special symbol to get the underlying schema
      if (prop === SCHEMA_SYMBOL) return schema;
      // Forward all property access to the schema
      const value = Reflect.get(schema, prop, schema);
      // If it's a function, bind it to the schema
      if (typeof value === "function") return value.bind(schema);
      return value;
    },
    // Make instanceof checks work
    getPrototypeOf() {
      return Object.getPrototypeOf(schema);
    },
    // Forward apply to our builder function
    apply(target, thisArg, args) {
      return builderFn(...args);
    },
  }) as S & BuilderFn;
};

const createBuilder = <Args extends z.ZodTypeAny, Returns extends z.ZodTypeAny>(
  argsSchema: Args,
  returnsSchema: Returns,
): CallableBuilder<Args, Returns> => {
  const argsProxy = createSchemaProxy(argsSchema, <A extends z.ZodTypeAny>(schema: A) =>
    createBuilder(schema, returnsSchema),
  );

  const returnsProxy = createSchemaProxy(
    returnsSchema,
    <R extends z.ZodTypeAny>(schema: R) => createBuilder(argsSchema, schema),
  );

  return {
    args: argsProxy,
    returns: returnsProxy,
  } as unknown as CallableBuilder<Args, Returns>;
};

/**
 * Creates a callable method schema with fluent API for defining argument and return types.
 *
 * NOTE: We provide our own `callable` utility instead of using Zod's `z.function()` because
 * Zod 4 changed `z.function()` to be a "function factory" for creating validated functions
 * at runtime, rather than a schema type. This makes it unsuitable for our use case where
 * we need a simple schema structure to define RPC method signatures for type extraction.
 * Our `callable` provides a clean fluent API similar to Zod 3's `z.function()` while
 * producing a simple `{ args, returns }` object that works well with TypeScript type inference.
 *
 * @example
 * ```typescript
 * // No args, void return (fire-and-forget)
 * const onClick = callable();
 *
 * // With args, void return
 * const setName = callable().args(z.string());
 *
 * // No args, with return value
 * const getData = callable().returns(z.number());
 *
 * // With args and return value
 * const calculate = callable().args(z.object({ x: z.number() })).returns(z.number());
 *
 * // Use in a methods schema
 * const buttonMethodsZ = {
 *   onClick: callable(),
 *   setLabel: callable().args(z.string()),
 *   getValue: callable().returns(z.number()),
 * };
 * ```
 */
export const callable = (): CallableBuilder => createBuilder(z.void(), z.void());

/** Checks if a Zod schema is a void type. */
export const isVoid = (schema: z.ZodTypeAny): schema is z.ZodVoid =>
  schema instanceof z.ZodVoid;
