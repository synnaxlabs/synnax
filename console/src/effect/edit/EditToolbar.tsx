import { useSelect } from "@/effect/selectors";
import { Slate } from "@/slate";

export const EditToolbar = ({ layoutKey }: { layoutKey: string }) => {
  const effect = useSelect(layoutKey);
  if (effect == null) return null;
  console.log("EFFECT", effect);
  return <Slate.Toolbar layoutKey={effect?.slate} name={effect.name} />;
};
