type Value = number | bigint;

export const sub = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a - BigInt(b)) as V;
  return (a - Number(b)) as V;
};

export const add = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a + BigInt(b)) as V;
  // @ts-expect-error
  return (a + Number(b)) as V;
};
