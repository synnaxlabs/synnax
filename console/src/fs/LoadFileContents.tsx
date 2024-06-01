// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Input, Text } from "@synnaxlabs/pluto";
import { open } from "@tauri-apps/plugin-dialog";
import { ReactElement } from "react";

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
      if (path == null) return;
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
