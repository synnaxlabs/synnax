import { memo } from "react";
import {
  useLayoutRenderer,
  useSelectLayoutRendererProps,
} from "@/features/layout";

export const LayoutContent = memo(({ contentKey }: { contentKey: string }) => {
  const props = useSelectLayoutRendererProps(contentKey);
  if (!props) return null;
  const renderer = useLayoutRenderer(props.type);
  return renderer ? renderer(props) : null;
});
