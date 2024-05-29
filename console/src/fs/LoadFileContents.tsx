import { propertiesZ } from "@/hardware/opc/device/types";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Input, Text } from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { ReactElement, useState } from "react";

export interface LoadFileContentsProps
  extends Input.Control<string>,
    Omit<Align.PackProps, "value" | "onChange"> {}

export const LoadFileContents = ({
  value,
  onChange,
  ...props
}: LoadFileContentsProps): ReactElement => {
  const path = value;
  const handleClick = () => {
    void (async () => {
      const path = await open({ directory: false });
      if (path == null) {
        return;
      }
      const contents = await readFile(path.path);
      const text = new TextDecoder().decode(contents);
      onChange(path.path);
    })();
  };

  return (
    <Align.Pack direction="x" {...props}>
      <Text.WithIcon
        level="p"
        style={{ padding: "0 2rem" }}
        shade={path == null ? 7 : 9}
        grow
        startIcon={
          path == null ? undefined : (
            <Icon.Attachment style={{ color: "var(--pluto-gray-l7)" }} />
          )
        }
        size="medium"
      >
        {path == null ? "No file selected" : path}
      </Text.WithIcon>
      <Button.Button variant="outlined" onClick={handleClick}>
        Select file
      </Button.Button>
    </Align.Pack>
  );
};
