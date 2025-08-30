import { Flex, Haul, Icon, Status, Text } from "@synnaxlabs/pluto";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { type ReactElement, useState } from "react";

import { CSS } from "@/css";

const canDrop: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;
export interface FileDropProps extends Flex.BoxProps {
  onContentsChange: (contents: string) => void;
  enabled?: boolean;
}

export const FileDrop = ({
  onContentsChange,
  children,
  enabled = true,
  ...rest
}: FileDropProps): ReactElement => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const [draggingOver, setDraggingOver] = useState(false);
  const handleFileDrop = ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
    if (event == null) return items;
    event.preventDefault();
    setDraggingOver(false);
    if (event.dataTransfer.files.length === 0) return items;

    const file = event.dataTransfer.files[0];
    if (!file.name.toLowerCase().endsWith(".svg")) {
      addStatus({ message: "Invalid file type", variant: "error" });
      return items;
    }

    handleError(async () => {
      const svg = await file.text();
      onContentsChange(svg);
    }, "Failed to load dropped SVG file");
    return items;
  };

  const handleFileSelect = () =>
    handleError(async () => {
      const path = await open({
        directory: false,
        filters: [{ name: "SVG Files", extensions: ["svg"] }],
      });
      if (path == null) return;
      const contents = await readTextFile(path);
      if (contents == null) return;
      onContentsChange(contents);
    }, "Failed to load SVG file");

  const dropProps = Haul.useDrop({
    type: Haul.FILE_TYPE,
    onDrop: handleFileDrop,
    canDrop,
    onDragOver: () => setDraggingOver(true),
  });
  return (
    <Flex.Box
      grow
      align="center"
      justify="center"
      bordered
      className={CSS(
        CSS.B("file-drop"),
        draggingOver && CSS.M("dragging-over"),
        enabled && CSS.M("enabled"),
      )}
      onDragLeave={() => setDraggingOver(false)}
      rounded={1}
      onClick={enabled ? handleFileSelect : undefined}
      {...dropProps}
      borderColor={5}
      {...rest}
      style={{ boxShadow: "var(--pluto-shadow-v2)" }}
    >
      {enabled && (
        <Flex.Box y align="center" center style={{ position: "absolute" }}>
          <Text.Text level="h1" color={7}>
            <Icon.Import />
          </Text.Text>
          <Text.Text level="p" color={9}>
            Click to select an SVG file or drag and drop it here
          </Text.Text>
        </Flex.Box>
      )}
      {children}
    </Flex.Box>
  );
};
