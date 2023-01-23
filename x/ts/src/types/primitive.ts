export type Primitive =
  | string
  | number
  | bigint
  | boolean
  | Stringer
  | null
  | undefined;

export interface Stringer {
  toString: () => string;
}

export const isStringer = (value: unknown): boolean =>
  value != null && typeof value === "object" && "toString" in value;

export type PrimitiveRecord = Record<string, Primitive>;
