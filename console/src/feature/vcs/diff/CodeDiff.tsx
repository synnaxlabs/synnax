// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { Code } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Target } from "@/feature/vcs/diff/Provider";

export interface CodeDiffProps {
  target: Target;
  kind: vcs.ChangeKind;
  before: vcs.File[];
  after: vcs.File[];
  /** extension picks the file of each side to compare, such as ".arc". */
  extension: string;
  language: string;
}

const label = (rev: string, file: vcs.File | undefined): string | undefined =>
  file == null ? undefined : `${rev} · ${file.path}`;

/** Compares one file of a resource side by side, or alone when only one side has it. */
export const CodeDiff = ({
  target,
  kind,
  before,
  after,
  extension,
  language,
}: CodeDiffProps): ReactElement => {
  const prev = before.find(({ path }) => path.endsWith(extension));
  const next = after.find(({ path }) => path.endsWith(extension));
  return (
    <Code.DiffEditor
      original={prev?.content ?? ""}
      modified={next?.content ?? ""}
      language={language}
      inline={kind !== "modified"}
      originalLabel={kind === "added" ? undefined : label(target.beforeLabel, prev)}
      modifiedLabel={kind === "deleted" ? undefined : label(target.afterLabel, next)}
      background={0}
    />
  );
};
