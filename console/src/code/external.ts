import { EDITOR_LAYOUT_TYPE,EditorLayout } from "@/code/Editor";
import { type Layout } from "@/layout";

export * from "@/code/Editor";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EDITOR_LAYOUT_TYPE]: EditorLayout,
};
