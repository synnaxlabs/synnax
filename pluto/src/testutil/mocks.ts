import { vi } from "vitest";

// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
export const mockBoundingClientRect = (
  top: number,
  left: number,
  width: number,
  height: number
) =>
  vi.fn().mockReturnValue({
    top,
    left,
    width,
    height,
    bottom: top + height,
    right: left + width,
    x: 0,
    y: 0,
    toJSON: () => "",
  });
