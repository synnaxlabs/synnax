export const toArray = <T>(value: T | T[]): T[] =>
  Array.isArray(value) ? value : [value];
