// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { assetURL } from "@/util/releases";

const SUFFIXES: Record<runtime.OS, string | null> = {
  macOS: "_aarch64.dmg",
  Windows: "_x64-setup.exe",
  Linux: null,
};

export interface DownloadLinkProps {
  version: string;
}

/** Links to the Console installer for the viewer's OS, or nothing on Linux. */
export const DownloadLink = ({ version }: DownloadLinkProps): ReactElement | null => {
  const os = runtime.getOS();
  const suffix = SUFFIXES[os];
  if (suffix == null) return null;
  const url = assetURL("console", version, `Synnax_${version}${suffix}`);
  return (
    <Button.Button href={url} variant="filled">
      <Icon.Download />
      Download v{version} for {os}
    </Button.Button>
  );
};
