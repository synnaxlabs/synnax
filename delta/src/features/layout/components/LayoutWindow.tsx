import { NavTop } from "@/components/MainLayout/NavTop";
import { Space } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import { useLayoutRenderer } from "../context";
import { useLayoutRemover } from "../hooks";
import { useSelectLayout } from "../store";

export const LayoutWindow = () => {
  const key = appWindow.label;
  const handleClose = useLayoutRemover(key);
  const layout = useSelectLayout(key);
  const Renderer = useLayoutRenderer(layout?.type);
  if (!Renderer) return null;

  const renderedContent = <Renderer layoutKey={key} onClose={handleClose} />;

  if (layout.window?.navTop) {
    return (
      <Space direction="vertical" empty style={{ height: "100vh" }}>
        <NavTop />
        {renderedContent}
      </Space>
    );
  }
  return renderedContent;
};
