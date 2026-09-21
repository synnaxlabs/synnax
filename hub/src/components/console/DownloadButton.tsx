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
import { type ReactElement, useEffect, useState } from "react";

interface PlatformEntry {
  url: string;
}

interface UpdateFile {
  version: string;
  platforms: {
    "darwin-x86_64": PlatformEntry;
    "linux-x86_64": PlatformEntry;
    "windows-x86_64": PlatformEntry;
  };
}

const SUFFIXES: Record<runtime.OS, string | null> = {
  macOS: "_aarch64.dmg",
  Windows: "_x64-setup.exe",
  Linux: null,
};

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/console/release-spec.json";

export const DownloadButton = (): ReactElement | null => {
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);
  useEffect(() => {
    fetch(JSON_URL)
      .then(async (response) => (await response.json()) as UpdateFile)
      .then(setUpdateFile)
      .catch(() => setUpdateFile(null));
  }, []);
  if (updateFile == null) return null;
  const version = updateFile.version;
  const baseURL = `https://github.com/synnaxlabs/synnax/releases/download/console-${version}/Synnax_${version.slice(1)}`;
  const os = runtime.getOS();
  const suffix = SUFFIXES[os];
  if (suffix == null) return null;
  return (
    <Button.Button href={`${baseURL}${suffix}`} variant="filled">
      <Icon.Download />
      Download {version} for {os}
    </Button.Button>
  );
};
