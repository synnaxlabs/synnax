// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/useImportGroup.css";

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Button, Flex, Haul, Icon, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { zipSync } from "fflate";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/platform/css";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

const canDrop: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;

const isZipFile = (name: string): boolean => name.toLowerCase().endsWith(".zip");

interface BundleFile {
  name: string;
  bytes: Uint8Array<ArrayBuffer>;
}

// Entries are stored uncompressed; symbol files are small.
const zipFiles = (files: BundleFile[]): Uint8Array<ArrayBuffer> =>
  zipSync(Object.fromEntries(files.map(({ name, bytes }) => [name, bytes])), {
    level: 0,
  });

const importBundle = async (client: Client | null, bundle: Uint8Array<ArrayBuffer>) => {
  if (client == null) throw new DisconnectedError();
  return await client.schematics.symbols.importGroup(bundle);
};

/**
 * A modal that imports a symbol group from a .zip export or its extracted folder:
 * drop either onto the zone, click it to browse for a .zip, or use the folder button
 * for a directory picker. The Core owns the bundle format, so a folder's top-level
 * files are zipped and uploaded untouched.
 */
export const useImportGroup = Modals.create(({ close }): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const [draggingOver, setDraggingOver] = useState(false);

  const importZip = useCallback(
    async (bundle: Uint8Array<ArrayBuffer>): Promise<void> => {
      const imported = await importBundle(client, bundle);
      addStatus({
        variant: "success",
        message: `Imported symbol group "${imported.name}"`,
      });
      close();
    },
    [client, addStatus, close],
  );

  const handlePickZip = (): void =>
    handleError(async () => {
      const files = await Runtime.pickFiles({
        title: "Import symbol group",
        filters: [{ name: "Zip", extensions: ["zip"] }],
      });
      if (files == null) return;
      await importZip(await files[0].readBytes());
    }, "Failed to import symbol group");

  const handlePickDirectory = (): void =>
    handleError(async () => {
      const directory = await Runtime.pickDirectory({ title: "Import symbol group" });
      if (directory == null) return;
      const files = directory.files.filter((file) => !file.path.includes("/"));
      await importZip(
        zipFiles(
          await Promise.all(
            files.map(async (file) => ({
              name: file.path,
              bytes: await file.readBytes(),
            })),
          ),
        ),
      );
    }, "Failed to import symbol group");

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      if (event == null) return items;
      event.preventDefault();
      setDraggingOver(false);
      const entries = Import.captureEntries(event.dataTransfer);
      handleError(async () => {
        if (entries.length !== 1) throw new Error("drop a single .zip file or folder");
        const [entry] = entries;
        if (Import.isDirectoryEntry(entry)) {
          const files = await Import.readDirectoryFiles(entry);
          await importZip(
            zipFiles(
              await Promise.all(
                files
                  .filter(({ path }) => !path.includes("/"))
                  .map(async ({ file, path }) => ({
                    name: path,
                    bytes: new Uint8Array(await file.arrayBuffer()),
                  })),
              ),
            ),
          );
          return;
        }
        if (!Import.isFileEntry(entry))
          throw new Error("dropped item is not a file or folder");
        const file = await Import.readEntryFile(entry);
        if (!isZipFile(file.name)) throw new Error(`${file.name} is not a .zip file`);
        await importZip(new Uint8Array(await file.arrayBuffer()));
      }, "Failed to import symbol group");
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
    <Modals.Frame className={CSS.BE("schematic", "symbol-import-group")}>
      <Modals.Header icon={<Icon.Import />}>
        Schematic.Symbols.Group.Import
      </Modals.Header>
      <Modals.Body>
        <Flex.Box
          className={CSS.BE("schematic", "symbol-import-group", "zone")}
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
