// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z, type ZodSchema } from "zod";

import { deep } from "@/deep";
import { type UnknownRecord } from "@/record";

export const getFieldSchemaPath = (path: string): string =>
  deep.transformPath(path, (part, index, parts) => {
    const isLast = index === parts.length - 1;
    const isNumericPart = !isNaN(parseInt(part));
    const nextPartIsNumeric = !isNaN(parseInt(parts[index + 1]));
    if (isNumericPart) part = "element";
    if (isLast || nextPartIsNumeric) return part;
    return [part, "shape"];
  });

const sourceTypeGetter = (obj: unknown, key: string): z.ZodAny | null => {
  if (obj == null) return null;
  const v = (obj as Record<string, z.ZodAny>)[key];
  if (v == null && typeof obj === "object" && "sourceType" in obj) {
    const sourceType = (obj as z.ZodEffects<z.ZodTypeAny>).sourceType();
    return (sourceType as unknown as UnknownRecord)[key] as z.ZodAny | null;
  }
  return v;
};

export const getFieldSchema: deep.TypedGet<z.ZodTypeAny, z.ZodTypeAny> = ((
  schema: z.ZodTypeAny,
  path: string,
  options?: Omit<deep.GetOptions, "getter">,
): z.ZodTypeAny | null =>
  deep.get<z.ZodTypeAny, z.ZodTypeAny>(
    sourceTypeGetter(schema, "shape") as unknown as z.AnyZodObject,
    getFieldSchemaPath(path),
    { ...options, getter: sourceTypeGetter } as deep.GetOptions<boolean | undefined>,
  ) as z.ZodTypeAny | null) as deep.TypedGet<z.ZodTypeAny, z.ZodTypeAny>;

/**
 * Creates a transformer function that validates and transforms input values based on
 * provided schemas. The first schema to successfully validate the input value is used
 * in the transformation. If no schema is found that validates the input, the
 * transformer function returns null.
 *
 * @template Input - The type of the input value.
 * @template Output - The type of the output value.
 * @param transform - The function to transform the input value to the output value.
 * @param schemas - An array of Zod schemas to validate the input value against.
 * @returns A function that takes an unknown value, validates it against the schemas,
 * and uses the first valid schema to transform the input type. If no schema can
 * validate the input, the function returns null.
 */
export const transformer =
  <Input, Output>(
    transform: (input: Input) => Output,
    schemas: ZodSchema<Input>[],
  ): ((value: unknown) => Output | null) =>
  (value) => {
    const matchingSchema = schemas.find((schema) => schema.safeParse(value).success);
    if (matchingSchema == null) return null;
    return transform(matchingSchema.parse(value));
  };

export const bigInt = z.bigint().or(z.string().transform(BigInt));
