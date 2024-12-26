// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

export interface OSDownloadButtonEntry {
  os: runtime.OS;
  href: string;
}

export interface OSDownloadButtonProps extends Omit<Button.LinkProps, "href"> {
  name: string;
  entries: OSDownloadButtonEntry[];
}

export const OSDownloadButton = ({
  entries = [],
  name,
  ...props
}: OSDownloadButtonProps): ReactElement | null => {
  const os = runtime.getOS();
  const entry = entries.find((entry) => entry.os === os);
  if (entry == null) return null;
  const { href } = entry;
  return (
    <Button.Link href={href} startIcon={<Icon.Download />} {...props}>
      Download {name} for {os}
    </Button.Link>
  );
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
  };
}

const OSToUpdateFilePlatform: Record<
  "MacOS" | "Linux" | "Windows",
  keyof UpdateFile["platforms"]
> = {
  MacOS: "darwin-x86_64",
  Linux: "linux-x86_64",
  Windows: "windows-x86_64",
};

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/console/release-spec.json";

export const SynnaxConsoleDownloadButton = (): ReactElement | null => {
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);

  useEffect(() => {
    fetch(JSON_URL)
      .then(async (response) => await response.json())
      .then((f) => setUpdateFile(f as UpdateFile))
      .catch(() => setUpdateFile(null));
  }, []);

  if (updateFile == null) return null;
  return (
    <OSDownloadButton
      className="os-download-button"
      name={updateFile.version}
      size="large"
      entries={[
        {
          os: "MacOS",
          href: `https://github.com/synnaxlabs/synnax/releases/download/console-${updateFile.version}/Synnax_${updateFile.version.slice(1)}_aarch64.dmg`,
        },
        {
          os: "Windows",
          href: updateFile.platforms[OSToUpdateFilePlatform.Windows].url,
        },
      ]}
    />
  );
};
