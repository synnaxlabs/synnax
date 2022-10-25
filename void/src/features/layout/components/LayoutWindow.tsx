import { appWindow } from "@tauri-apps/api/window";
import { useLayoutRenderer } from "../context";
import { useSelectLayoutContent, useSelectWindowPlacement } from "../store";

export const LayoutWindow = () => {
  const key = appWindow.label;
  const placement = useSelectWindowPlacement(key);
  if (!placement) return null;
  const content = useSelectLayoutContent(placement.contentKey);
  const renderer = useLayoutRenderer(content.type);
  return renderer ? renderer(content) : null;
};
