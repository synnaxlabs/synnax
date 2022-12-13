import { ResizeObserver } from "@juggle/resize-observer";
import { afterAll, beforeAll, vi } from "vitest";

beforeAll(() => {
  vi.stubGlobal("ResizeObserver", ResizeObserver);
});

afterAll(() => {
  vi.clearAllMocks();
});
