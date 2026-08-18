// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/import/createModal.css";

import { Button, Icon, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/platform/css";
import { DropZone } from "@/platform/import/DropZone";
import {
  isDirectoryEntry,
  isFileEntry,
  readDirectoryFiles,
  readEntryFile,
} from "@/platform/import/entries";
import { isZipFile, zipFiles } from "@/platform/import/zip";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

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
        await importZip(await zipFiles(directory.files), directory.name);
      }, errorMessage);

    const handleDrop = useCallback(
      (entries: FileSystemEntry[]): void =>
        handleError(async () => {
          if (entries.length !== 1)
            throw new Error("drop a single .zip file or folder");
          const [entry] = entries;
          if (isDirectoryEntry(entry)) {
            await importZip(
              await zipFiles(await readDirectoryFiles(entry)),
              entry.name,
            );
            return;
          }
          if (!isFileEntry(entry))
            throw new Error("dropped item is not a file or folder");
          const file = await readEntryFile(entry);
          if (!isZipFile(file.name)) throw new Error(`${file.name} is not a .zip file`);
          await importZip(new Uint8Array(await file.arrayBuffer()), file.name);
        }, errorMessage),
      [handleError, importZip],
    );

    return (
      <Modals.Frame className={CSS.B("import-modal")}>
        <Modals.Header icon={<Icon.Import />}>{header}</Modals.Header>
        <Modals.Body>
          <DropZone
            className={CSS.BE("import-modal", "zone")}
            y
            gap="small"
            onDrop={handleDrop}
            onClick={handlePickZip}
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
          </DropZone>
        </Modals.Body>
      </Modals.Frame>
    );
  });
