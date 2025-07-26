import { useSelect } from "@/effect/selectors";
import { Layout } from "@/layout";
import { Slate } from "@/slate";

export const EditToolbar = ({ layoutKey }: { layoutKey: string }) => {
  const effect = useSelect(layoutKey);
  const layout = Layout.useSelect(layoutKey);
  if (effect == null || layout == null) return null;
  return <Slate.Toolbar layoutKey={effect?.slate} name={layout.name} />;
};
