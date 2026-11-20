// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/certificate/PEMInput.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { FS } from "@/platform/fs";
import { Runtime } from "@/platform/runtime";

const READ_FAILED = "Failed to read file";

export interface PEMInputProps
  extends Input.Control<string>, Pick<Input.TextProps, "placeholder" | "autoFocus"> {
  /** Titles the file dialog. */
  title: string;
  /** The extensions the file dialog offers, without leading dots. */
  extensions: string[];
  /** Reads a dropped or selected file as PEM. */
  read: (bytes: Uint8Array, name: string) => string;
}

/** A text area for pasted PEM that also takes a dropped or selected file. */
export const PEMInput = ({
  onChange,
  title,
  extensions,
  read,
  ...rest
}: PEMInputProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const handleDrop = (entries: FileSystemEntry[]): void =>
    handleError(async () => {
      const [entry] = entries;
      if (entries.length !== 1 || !FS.isFileEntry(entry))
        throw new Error("Drop a single file.");
      const file = await FS.readEntryFile(entry);
      onChange(read(await Runtime.toBytes(file), file.name));
    }, READ_FAILED);
  const handleSelect = (): void =>
    handleError(async () => {
      const file = await Runtime.pickFiles({ title, extension: extensions });
      if (file == null) return;
      onChange(read(await Runtime.toBytes(await file.read()), file.path));
    }, READ_FAILED);
  return (
    <FS.DropZone
      y
      align="stretch"
      justify="start"
      className={CSS.B("pem-input")}
      onDrop={handleDrop}
    >
      <Input.Text
        area
        variant="text"
        onChange={onChange}
        spellCheck={false}
        autoComplete="off"
        className={CSS.BE("pem-input", "text")}
        {...rest}
      />
      <Flex.Box
        x
        align="center"
        justify="between"
        className={CSS.BE("pem-input", "footer")}
      >
        <Text.Text level="small" color={8}>
          Paste PEM or drop a file
        </Text.Text>
        <Button.Button variant="text" size="small" textColor={9} onClick={handleSelect}>
          <Icon.Import />
          Select file
        </Button.Button>
      </Flex.Box>
    </FS.DropZone>
  );
};
