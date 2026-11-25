// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { group } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { Group } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

/** @returns the names of the folders from the top down to the folder with the key. */
const folderNames = (key: group.Key, folders: group.Group[]): string[] => {
  const byKey = new Map(folders.map((f) => [f.key, f]));
  const names: string[] = [];
  let folder = byKey.get(key);
  while (folder != null) {
    names.unshift(folder.name);
    const parent = folder.parent;
    folder = parent?.type === "group" ? byKey.get(parent.key) : undefined;
  }
  return names;
};

export interface FolderPathProps {
  folder: group.Key;
}

/** Shows the folders that hold a channel, and follows a rename or move of each. */
export const FolderPath = ({ folder }: FolderPathProps): ReactElement | null => {
  const folders = Group.useResultAncestors({ keys: [folder] }).data;
  if (folders == null) return null;
  return (
    <Text.Text level="small" color={9} overflow="fade">
      {folderNames(folder, folders).join(" / ")}
    </Text.Text>
  );
};
