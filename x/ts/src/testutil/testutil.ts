export const toString = (value: unknown): string =>
  JSON.stringify(value, (_, value) => {
    if (typeof value === "bigint") return value.toString();
    return value;
  });
