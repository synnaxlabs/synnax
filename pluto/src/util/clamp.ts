export const clamp = (value: number, min?: number, max?: number): number => {
  if (min != null) value = Math.max(value, min);
  if (max != null) value = Math.min(value, max);
  return value;
};
