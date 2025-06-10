// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/fs/LoadFileContents.css";

import { Icon } from "@synnaxlabs/media";
import { Align, Button, type Input, Status } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { type DialogFilter, open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { type ReactElement, useEffect, useState } from "react";
import { type z } from "zod/v4";

import { CSS } from "@/css";

export interface InputFilePathProps
  extends Input.Control<string>,
    Omit<Align.PackProps, "value" | "onChange"> {
  filters?: DialogFilter[];
}

export const InputFilePath = ({
  value,
  onChange,
  filters,
  ...rest
}: InputFilePathProps): ReactElement => {
  const path = value;
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(async () => {
      const path = await open({ directory: false, filters });
      if (path == null) return;
      onChange(path);
    }, "Failed to open file");
  return (
    <Align.Pack className={CSS.B("input-file-path")} borderShade={5} {...rest}>
      <Button.Button
        level="small"
        className={CSS.B("path")}
        variant="outlined"
        shade={0}
        grow
        onClick={handleClick}
        startIcon={
          path == null ? undefined : (
            <Icon.Attachment style={{ color: "var(--pluto-gray-l8)" }} />
          )
        }
        size="medium"
        textShade={8}
        weight={450}
      >
        {path == null ? "No file selected" : path}
      </Button.Button>
      <Button.Button
        variant="outlined"
        className={CSS.B("select")}
        onClick={handleClick}
      >
        Select file
      </Button.Button>
    </Align.Pack>
  );
};

export interface InputFileContentsProps<P extends z.ZodType = z.ZodString>
  extends Omit<InputFilePathProps, "value" | "onChange"> {
  onChange: (value: z.infer<P>, path: string) => void;
  initialPath?: string;
  schema?: P;
  decoder?: binary.Codec;
}

export const InputFileContents = <P extends z.ZodType = z.ZodString>({
  onChange,
  decoder = binary.TEXT_CODEC,
  initialPath,
  schema,
  ...rest
}: InputFileContentsProps<P>): ReactElement => {
  const handleError = Status.useErrorHandler();
  const [path, setPath] = useState<string>("");
  useEffect(() => {
    if (initialPath == null || initialPath === path) return;
    handleChange(initialPath);
  }, [initialPath]);
  const handleChange = (path: string) =>
    handleError(async () => {
      const contents = await readFile(path);
      if (contents == null) return;
      onChange(decoder.decode<P>(contents, schema), path);
      setPath(path);
    }, "Failed to read file");
  return <InputFilePath value={path} onChange={handleChange} {...rest} />;
};
