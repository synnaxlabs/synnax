import { memo } from "react";

import { useLayoutRenderer } from "../context";
import { useSelectLayout } from "../store";

export const LayoutContent = memo(
  ({ layoutKey }: { layoutKey: string }): JSX.Element | null => {
    const p = useSelectLayout(layoutKey);
    if (p == null) throw new Error(`layout ${layoutKey} not found`);
    if (p.type == null) throw new Error("layout has no type");
    const Renderer = useLayoutRenderer(p.type);
    if (Renderer == null) throw new Error(`layout renderer ${p.type} not found`);
    return <Renderer layoutKey={layoutKey} onClose={() => {}} />;
  }
);
LayoutContent.displayName = "LayoutContent";
