// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/import/createModal.css";

import { Button, Flex, Haul, Icon, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/platform/css";
import {
  captureEntries,
  isDirectoryEntry,
  isFileEntry,
  readDirectoryFiles,
  readEntryFile,
} from "@/platform/import/entries";
import { isZipFile, zipFiles } from "@/platform/import/zip";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

const canDrop: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;

export interface CreateModalArgs {
  /** The modal's header text, e.g. "Project.Import". */
  header: string;
  /** Names the imported kind in picker titles and statuses, e.g. "project". */
  resourceName: string;
  /**
   * A hook returning the import call: it uploads the zipped bundle named by the source
   * archive or folder's name and returns the imported resource's name for the success
   * status.
   */
  useOnImport: () => (
    bundle: Uint8Array<ArrayBuffer>,
    fileName: string,
  ) => Promise<string>;
}

/**
 * Creates a modal hook that imports a bundle from a .zip export or its extracted
 * folder: drop either onto the zone, click it to browse for a .zip, or use the folder
 * button for a directory picker. The Core owns the bundle format, so a folder's files
 * are zipped with their relative paths and uploaded untouched.
 */
export const createModal = ({ header, resourceName, useOnImport }: CreateModalArgs) =>
  Modals.create(({ close }): ReactElement => {
    const onImport = useOnImport();
    const handleError = Status.useErrorHandler();
    const addStatus = Status.useAdder();
    const [draggingOver, setDraggingOver] = useState(false);
    const errorMessage = `Failed to import ${resourceName}`;

    const importZip = useCallback(
      async (bundle: Uint8Array<ArrayBuffer>, fileName: string): Promise<void> => {
        const name = await onImport(bundle, fileName);
        addStatus({
          variant: "success",
          message: `Imported ${resourceName} "${name}"`,
        });
        close();
      },
      [onImport, addStatus, close],
    );

    const handlePickZip = (): void =>
      handleError(async () => {
        const files = await Runtime.pickFiles({
          title: `Import ${resourceName}`,
          filters: [{ name: "Zip", extensions: ["zip"] }],
        });
        if (files == null) return;
        await importZip(await files[0].readBytes(), files[0].name);
      }, errorMessage);

    const handlePickDirectory = (): void =>
      handleError(async () => {
        const directory = await Runtime.pickDirectory({
          title: `Import ${resourceName}`,
        });
        if (directory == null) return;
        await importZip(
          zipFiles(
            await Promise.all(
              directory.files.map(async (file) => ({
                path: file.path,
                bytes: await file.readBytes(),
              })),
            ),
          ),
          directory.name,
        );
      }, errorMessage);

    const handleDrop = useCallback(
      ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
        if (event == null) return items;
        event.preventDefault();
        setDraggingOver(false);
        const entries = captureEntries(event.dataTransfer);
        handleError(async () => {
          if (entries.length !== 1)
            throw new Error("drop a single .zip file or folder");
          const [entry] = entries;
          if (isDirectoryEntry(entry)) {
            const files = await readDirectoryFiles(entry);
            await importZip(
              zipFiles(
                await Promise.all(
                  files.map(async ({ file, path }) => ({
                    path,
                    bytes: new Uint8Array(await file.arrayBuffer()),
                  })),
                ),
              ),
              entry.name,
            );
            return;
          }
          if (!isFileEntry(entry))
            throw new Error("dropped item is not a file or folder");
          const file = await readEntryFile(entry);
          if (!isZipFile(file.name)) throw new Error(`${file.name} is not a .zip file`);
          await importZip(new Uint8Array(await file.arrayBuffer()), file.name);
        }, errorMessage);
        return items;
      },
      [handleError, importZip],
    );

    const dropProps = Haul.useDrop({
      type: Haul.FILE_TYPE,
      onDrop: handleDrop,
      canDrop,
      onDragOver: () => setDraggingOver(true),
    });

    return (
      <Modals.Frame className={CSS.B("import-modal")}>
        <Modals.Header icon={<Icon.Import />}>{header}</Modals.Header>
        <Modals.Body>
          <Flex.Box
            className={CSS.BE("import-modal", "zone")}
            y
            align="center"
            justify="center"
            gap="small"
            bordered
            rounded="small"
            borderColor={draggingOver ? 9 : 6}
            onDragLeave={() => setDraggingOver(false)}
            onClick={handlePickZip}
            {...dropProps}
          >
            <Text.Text level="h1" color={9}>
              <Icon.Import />
            </Text.Text>
            <Text.Text level="p" color={9}>
              Drop a .zip or folder here
            </Text.Text>
            <Text.Text level="small" color={8}>
              or click to browse
            </Text.Text>
            <Button.Button
              variant="text"
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                handlePickDirectory();
              }}
            >
              Select folder
            </Button.Button>
          </Flex.Box>
        </Modals.Body>
      </Modals.Frame>
    );
  });
