import { Editor, EDITOR_LAYOUT_TYPE } from "@/code/Editor";
import { Layout } from "@/layout";

export * from "@/code/Editor";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EDITOR_LAYOUT_TYPE]: Editor,
};