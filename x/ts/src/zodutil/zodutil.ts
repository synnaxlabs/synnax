import { z } from "zod";

import { deep } from "@/deep";
import { UnknownRecord } from "@/record";

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
  allowNull?: boolean,
): z.ZodTypeAny | null =>
  deep.get<z.ZodTypeAny, z.ZodTypeAny>(
    sourceTypeGetter(schema, "shape") as unknown as z.AnyZodObject,
    getFieldSchemaPath(path),
    allowNull,
    { getter: sourceTypeGetter },
  ) as z.ZodTypeAny | null) as deep.TypedGet<z.ZodTypeAny, z.ZodTypeAny>;
