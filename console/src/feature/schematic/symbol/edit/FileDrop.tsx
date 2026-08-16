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
import { Flex, Haul, Icon, Status, Text } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";

const canDrop: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;

const isSVGFile = (name: string): boolean => name.toLowerCase().endsWith(".svg");

const INVALID_FILE_TYPE_STATUS: status.Crude = {
  variant: "error",
  message: "Invalid file type. Expected an SVG file.",
};

export interface FileDropProps extends Flex.BoxProps {
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
  const [draggingOver, setDraggingOver] = useState(false);
  const handleFileDrop = ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
    if (event == null) return items;
    event.preventDefault();
    setDraggingOver(false);
    if (event.dataTransfer.files.length === 0) return items;

    const file = event.dataTransfer.files[0];
    if (!isSVGFile(file.name)) {
      addStatus(INVALID_FILE_TYPE_STATUS);
      return items;
    }

    handleError(async () => {
      const svg = await file.text();
      const nameWithoutExt = file.name.replace(/\.svg$/i, "");
      const properName = caseconv.toProperNoun(nameWithoutExt);
      onContentsChange(svg, properName);
    }, "Failed to load dropped SVG file");
    return items;
  };

  const handleFileSelect = () =>
    handleError(async () => {
      const files = await Runtime.pickFiles({
        filters: [{ name: "SVG Files", extensions: ["svg"] }],
      });
      if (files == null) return;
      const [file] = files;
      if (!isSVGFile(file.name)) {
        addStatus(INVALID_FILE_TYPE_STATUS);
        return;
      }
      const contents = await file.read();
      const nameWithoutExt = file.name.replace(/\.svg$/i, "");
      const properName =
        nameWithoutExt === "" ? undefined : caseconv.toProperNoun(nameWithoutExt);
      onContentsChange(contents, properName);
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
        CSS.B("schematic-file-drop"),
        draggingOver && CSS.M("dragging-over"),
        enabled && CSS.M("enabled"),
      )}
      onDragLeave={() => setDraggingOver(false)}
      rounded="small"
      onClick={enabled ? handleFileSelect : undefined}
      {...dropProps}
      borderColor={6}
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
    </Flex.Box>
  );
};
