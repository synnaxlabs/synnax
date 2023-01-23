// eslint-disable-next-line @typescript-eslint/no-explicit-any

export const debounce = <F extends (...args: any[]) => void>(
  func: F,
  waitFor: number
): F => {
  let timeout: ReturnType<typeof setTimeout> | null = null;
  if (waitFor === 0) return func;

  const debounced = (...args: Parameters<F>): void => {
    if (timeout !== null) {
      clearTimeout(timeout);
      timeout = null;
    }
    timeout = setTimeout(() => func(...args), waitFor);
  };

  return debounced as F;
};
