export type Value = number | bigint;
export type ValueGuard<T> = [T] extends [number]
  ? number
  : [T] extends [bigint]
    ? bigint
    : never;
