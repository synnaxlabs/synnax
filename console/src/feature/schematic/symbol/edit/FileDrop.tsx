// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { type status } from "@synnaxlabs/client";
import { Flex, Icon, Status, Text } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Import } from "@/platform/import";
import { Runtime } from "@/platform/runtime";

const isSVGFile = (name: string): boolean => name.toLowerCase().endsWith(".svg");

const INVALID_FILE_TYPE_STATUS: status.Crude = {
  variant: "error",
  message: "Invalid file type. Expected an SVG file.",
};

export interface FileDropProps extends Omit<Flex.BoxProps, "onDrop" | "onClick"> {
  onContentsChange: (contents: string, filename?: string) => void;
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

  const loadSVG = (name: string, read: () => Promise<string>): void => {
    if (!isSVGFile(name)) {
      addStatus(INVALID_FILE_TYPE_STATUS);
      return;
    }
    handleError(async () => {
      const contents = await read();
      const nameWithoutExt = name.replace(/\.svg$/i, "");
      const properName =
        nameWithoutExt === "" ? undefined : caseconv.toProperNoun(nameWithoutExt);
      onContentsChange(contents, properName);
    }, "Failed to load SVG file");
  };

  const handleDrop = (entries: FileSystemEntry[]): void => {
    const [entry] = entries;
    if (entry == null || !Import.isFileEntry(entry)) return;
    handleError(async () => {
      const file = await Import.readEntryFile(entry);
      loadSVG(file.name, () => file.text());
    }, "Failed to load dropped SVG file");
  };

  const handleFileSelect = () =>
    handleError(async () => {
      const files = await Runtime.pickFiles({
        filters: [{ name: "SVG files", extensions: ["svg"] }],
      });
      if (files == null) return;
      const [file] = files;
      loadSVG(file.path, async () => new TextDecoder().decode(await file.readBytes()));
    }, "Failed to load SVG file");

  return (
    <Import.DropZone
      grow
      className={CSS(
        CSS.B("file-drop"),
        CSS.B("schematic-file-drop"),
        enabled && CSS.M("enabled"),
      )}
      onDrop={handleDrop}
      onClick={enabled ? handleFileSelect : undefined}
      {...rest}
    >
      {enabled && (
        <Flex.Box
          y
          align="center"
          center
          className={CSS.B("schematic-file-drop-overlay")}
        >
          <Text.Text level="h1" color={9}>
            <Icon.Import />
          </Text.Text>
          <Text.Text level="p" color={9}>
            Click to select an SVG file or drag and drop it here
          </Text.Text>
        </Flex.Box>
      )}
      {children}
    </Import.DropZone>
  );
};
