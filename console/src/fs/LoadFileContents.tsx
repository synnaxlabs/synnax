// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Input } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { ReactElement, useEffect, useState } from "react";
import { z } from "zod";

export interface InputFilePathProps
  extends Input.Control<string>,
    Omit<Align.PackProps, "value" | "onChange"> {}

export const InputFilePath = ({
  value,
  onChange,
  ...props
}: InputFilePathProps): ReactElement => {
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
      <Button.Button
        level="p"
        style={{ padding: "0 1.25rem", background: "var(--pluto-gray-l1)" }}
        variant="outlined"
        shade={path == null ? 7 : 9}
        grow
        onClick={handleClick}
        startIcon={
          path == null ? undefined : (
            <Icon.Attachment style={{ color: "var(--pluto-gray-l6)" }} />
          )
        }
        size="medium"
      >
        {path == null ? "No file selected" : path}
      </Button.Button>
      <Button.Button
        variant="outlined"
        onClick={handleClick}
        style={{ background: "var(--pluto-gray-l1)" }}
      >
        Select file
      </Button.Button>
    </Align.Pack>
  );
};

export interface InputFileContentsProps<P extends z.ZodTypeAny = z.ZodString>
  extends Omit<InputFilePathProps, "value" | "onChange"> {
  onChange: (value: z.output<P>, path: string) => void;
  initialPath?: string;
  schema?: P;
  decoder?: binary.EncoderDecoder;
}

export const InputFileContents = <P extends z.ZodTypeAny = z.ZodString>({
  onChange,
  decoder = binary.TEXT_ECD,
  initialPath,
  schema,
  ...props
}: InputFileContentsProps<P>): ReactElement => {
  const [path, setPath] = useState<string>("");
  useEffect(() => {
    if (initialPath == null || initialPath === path) return;
    handleChange(initialPath);
  }, [initialPath]);
  const handleChange = (path: string) => {
    void (async () => {
      const contents = await readFile(path);
      if (contents == null) return;
      onChange(decoder.decode<P>(contents, schema), path);
      setPath(path);
    })();
  };
  return <InputFilePath value={path} onChange={handleChange} {...props} />;
};
