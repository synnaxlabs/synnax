export const sortFunc = <V extends unknown>(
  t: string
): ((a: V, b: V) => number) => {
  switch (t) {
    case "string":
      return (a: V, b: V) => (a as string).localeCompare(b as string);
    case "number":
      return (a: V, b: V) => (a as number) - (b as number);
    default:
      console.warn("sortFunc: unknown type");
      return (a: V, b: V) => 0;
  }
};
