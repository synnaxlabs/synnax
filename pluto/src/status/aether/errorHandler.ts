import { type errors, status } from "@synnaxlabs/x";

export interface Adder {
  <D = undefined>(spec: status.Crude<D>): void;
}

export interface ErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): void;
  (exc: unknown, message?: string, skip?: errors.Matchable | errors.Matchable[]): void;
}

export interface AsyncErrorHandler {
  (
    func: () => Promise<void> | void,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
  (
    exc: unknown,
    message?: string,
    skip?: errors.Matchable | errors.Matchable[],
  ): Promise<void>;
}

const checkSkip = (
  err: unknown,
  skip: errors.Matchable | errors.Matchable[] | undefined,
): boolean => {
  if (Array.isArray(skip)) return skip.some((matcher) => matcher.matches(err));
  return skip?.matches(err) ?? false;
};

const parseException = (
  exc: unknown,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): status.Status | null => {
  const stat = status.fromException(exc, message);
  console.error(stat);
  if (checkSkip(exc, skip)) return null;
  return stat;
};

const handleException = <ExcOrFunc>(
  excOrFunc: ExcOrFunc,
  add: Adder,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): excOrFunc is ExcOrFunc & (() => Promise<void> | void) => {
  if (typeof excOrFunc === "function") return true;
  const stat = parseException(excOrFunc, message, skip);
  if (stat != null) add(stat);
  return false;
};

const handleFunc = async <Func extends () => Promise<void> | void>(
  func: Func,
  add: Adder,
  message?: string,
  skip?: errors.Matchable | errors.Matchable[],
): Promise<void> => {
  try {
    const promise = func();
    // Skip the added microtask if the function returns void instead of a promise.
    if (promise != null) await promise;
  } catch (exc) {
    const stat = parseException(exc, message, skip);
    if (stat != null) return add(stat);
  }
};

export const createErrorHandler =
  (add: Adder): ErrorHandler =>
  (excOrFunc, message, skip): void => {
    if (!handleException(excOrFunc, add, message, skip)) return;
    void handleFunc(excOrFunc, add, message, skip);
  };

export const createAsyncErrorHandler =
  (add: Adder): AsyncErrorHandler =>
  async (func, message, skip): Promise<void> => {
    if (!handleException(func, add, message, skip)) return;
    await handleFunc(func, add, message, skip);
  };
