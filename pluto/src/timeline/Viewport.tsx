import { useContext } from "@/timeline/context";
import { Viewport as Core } from "@/viewport";

export const Viewport = () => {
  const { viewport, setViewport } = useContext("Timeline.Viewport");

  const maskProps = Core.use({
    onChange: (e: Core.UseEvent) => setViewport(e.box),
    initial: viewport,
  });

  return <Core.Mask {...maskProps} />;
};
