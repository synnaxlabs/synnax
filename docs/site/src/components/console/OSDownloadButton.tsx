// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState, useEffect } from "react";

import { Button } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";

const OSToUpdateFilePlatform: Record<
  "MacOS" | "Linux" | "Windows" | "Docker",
  keyof UpdateFile["platforms"]
> = {
  MacOS: "darwin-x86_64",
  Linux: "linux-x86_64",
  Windows: "windows-x86_64",
  Docker: "docker",
};

export interface OSDownloadButtonEntry {
  os: runtime.OS;
  href: string;
  version: string;
}

export interface OSDownloadButtonProps extends Omit<Button.LinkProps, "href"> {
  name: string;
  entries: OSDownloadButtonEntry[];
}

export const useConsoleDownloadHref = (): OSDownloadButtonEntry | null => {
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);
  useEffect(() => {
    fetch(JSON_URL)
      .then(async (response) => await response.json())
      .then((f) => setUpdateFile(f as UpdateFile))
      .catch(() => setUpdateFile(null));
  }, []);
  if (updateFile == null) return null;
  const os = runtime.getOS();
  const platform = OSToUpdateFilePlatform[os];
  const href = updateFile.platforms[platform].url;
  return { os, href, version: updateFile.version };
};

export interface UpdateFile {
  version: string;
  platforms: {
    "darwin-x86_64": {
      url: string;
    };
    "linux-x86_64": {
      url: string;
    };
    "windows-x86_64": {
      url: string;
    };
    docker: {
      url: string;
    };
  };
}

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/console/release-spec.json";

export const SynnaxConsoleDownloadButton = (): ReactElement | null => {
  const entry = useConsoleDownloadHref();
  return (
    <Button.Link className="os-download-button" href={entry?.href}>
      Download Console {entry?.version} for {entry?.os}
    </Button.Link>
  );
};
