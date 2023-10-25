// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState, useEffect } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button } from "@synnaxlabs/pluto/button";
import { runtime } from "@synnaxlabs/x";

export interface OSDownloadButtonEntry {
  os: OS;
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
  // const os = useOS();
  if (entries.length === 0) return null;
  let entry = entries.find((entry) => entry.os === runtime.getOS());
  if (entry == null) entry = entries[0];
  const { href } = entry;
  return (
    <Button.Link href={href} startIcon={<Icon.Download />} {...props}>
      Download {name} for {runtime.getOS()}
    </Button.Link>
  );
};

export interface UpdateFile {
  version: string;
  platforms: {
    "darwin-x86_64": {
      href: string;
    };
    "linux-x86_64": {
      href: string;
    };
    "windows-x86_64": {
      href: string;
    };
  };
}

const OSToUpdateFilePlatform: Record<OS, keyof UpdateFile["platforms"]> = {
  MacOS: "darwin-x86_64",
  Linux: "linux-x86_64",
  Windows: "windows-x86_64",
};

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/delta/release-spec.json";

export const SynnaxConsoleDownloadButton = () => {
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);

  useEffect(() => {
    fetch(JSON_URL).then((response) => response.json()).then((f) => setUpdateFile(f))
  }, []);

  if (updateFile == null) return null;
  return (
    <OSDownloadButton
      name={updateFile.version}
      size="large"
      entries={[
        {
          os: "MacOS",
          href: updateFile.platforms[OSToUpdateFilePlatform.MacOS].url,
        },
        {
          os: "Linux",
          href: updateFile.platforms[OSToUpdateFilePlatform.Linux].url,
        },
        {
          os: "Windows",
          href: updateFile.platforms[OSToUpdateFilePlatform.Windows].url,
        },
      ]}
    />
  );
};
