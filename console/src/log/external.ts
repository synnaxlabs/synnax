import { Layout } from "@/layout";
import { LAYOUT_TYPE, Log, SELECTABLE } from "@/log/Log";

export * from "@/log/Log";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Log,
};

export const SELECTABLES: Layout.Selectable[] = [SELECTABLE];
