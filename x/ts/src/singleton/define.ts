const isDefined = (key: symbol): boolean =>
  Object.getOwnPropertySymbols(globalThis).includes(key);

/**
 * Defines a new global singleton instance of a value.
 *
 * @param key - The unique identifier for the singleton.
 * @param value - A function that returns the singleton instance.
 * @returns A function that returns the singleton instance.
 */
export const define = <T>(key: string, value: () => T): (() => T) => {
  const symbol = Symbol.for(key);
  if (!isDefined(symbol)) {
    const singleton = value();
    Object.defineProperty(globalThis, symbol, { value: singleton });
  }
  return () => (globalThis as any)[symbol] as T;
};
