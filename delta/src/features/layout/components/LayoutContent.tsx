import { memo } from "react";
import { useLayoutRenderer } from "../context";
import { useSelectLayout } from "../store";

export const LayoutContent = memo(({ layoutKey }: { layoutKey: string }) => {
	const props = useSelectLayout(layoutKey);
	const Renderer = useLayoutRenderer(props?.type);
	return Renderer ? <Renderer layoutKey={layoutKey} onClose={() => {}} /> : null;
});
