type Value = number | bigint;

/** @returns the product of a and b, coercing b to the type of a if necessary. */
export const sub = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a - BigInt(b)) as V;
  return (a - Number(b)) as V;
};

/** @returns the sum of a and b, coercing b to the type of a if necessary. */
export const add = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a + BigInt(b)) as V;
  // @ts-expect-error
  return (a + Number(b)) as V;
};

/** @returns true if a is close to b within epsilon. */
export const closeTo = (a: number, b: number, epsilon = 0.0001): boolean =>
  Math.abs(a - b) < epsilon;
