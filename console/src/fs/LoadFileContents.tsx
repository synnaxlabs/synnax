// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/fs/LoadFileContents.css";

import { Button, Flex, Icon, type Input, Status } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { type DialogFilter, open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { type ReactElement, useEffect, useState } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Runtime } from "@/runtime";

export interface InputFilePathProps
  extends Input.Control<string>, Omit<Flex.BoxProps, "value" | "onChange"> {
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
      if (Runtime.ENGINE !== "tauri")
        throw new Error(
          "Cannot open a file dialog when running Synnax in the browser.",
        );
      const path = await open({ directory: false, filters });
      if (path == null) return;
      onChange(path);
    }, "Failed to open file");
  return (
    <Flex.Box pack className={CSS.B("input-file-path")} borderColor={5} {...rest}>
      <Button.Button
        level="small"
        className={CSS.B("path")}
        variant="outlined"
        grow
        onClick={handleClick}
        size="medium"
        textColor={8}
        weight={450}
      >
        {path == null ? (
          "No file selected"
        ) : (
          <>
            <Icon.Attachment style={{ color: "var(--pluto-gray-l8)" }} />
            {path}
          </>
        )}
      </Button.Button>
      <Button.Button
        variant="outlined"
        className={CSS.B("select")}
        onClick={handleClick}
      >
        Select file
      </Button.Button>
    </Flex.Box>
  );
};

export interface InputFileContentsProps<P extends z.ZodType = z.ZodString> extends Omit<
  InputFilePathProps,
  "value" | "onChange"
> {
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
