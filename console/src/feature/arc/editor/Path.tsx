// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/Path.css";

import { type arc } from "@synnaxlabs/client";
import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";

// foldersOf returns the directories from below the root down to the directory with the
// key, root side first.
const foldersOf = (
  directories: arc.directory.Directory[],
  key: arc.directory.Key,
): arc.directory.Directory[] => {
  const byKey = new Map(directories.map((d) => [d.key, d]));
  const folders: arc.directory.Directory[] = [];
  let d = byKey.get(key);
  while (d?.parent != null) {
    folders.unshift(d);
    d = byKey.get(d.parent);
  }
  return folders;
};

/** The directories that hold the Arc, then the Arc, as a read-only breadcrumb. */
export const Path = (): ReactElement => {
  const name = Arc.useName();
  const TypeIcon = PlatformArc.TYPE_ICONS[Arc.useType()];
  const directory = Arc.useDirectory();
  const ancestors = Arc.Directory.useResultAncestors(
    directory == null ? null : { keys: [directory] },
  );
  const folders =
    directory != null && ancestors.variant === "success"
      ? foldersOf(ancestors.data, directory)
      : [];
  return (
    <Breadcrumb.Breadcrumb className={CSS.BE("arc-editor", "path")} level="small">
      {folders.map(({ key, name }) => (
        <Breadcrumb.Segment key={key} color={8}>
          {name}
        </Breadcrumb.Segment>
      ))}
      <Breadcrumb.Segment color={8} gap="small">
        <TypeIcon />
        {name}
      </Breadcrumb.Segment>
    </Breadcrumb.Breadcrumb>
  );
};
