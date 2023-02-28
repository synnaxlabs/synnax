// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, ButtonLinkProps, useOS, useAsyncEffect } from "@synnaxlabs/pluto";
import { OS } from "@synnaxlabs/x";

export interface OSDownloadButtonEntry {
  os: OS;
  href: string;
}

export interface OSDownloadButtonProps extends Omit<ButtonLinkProps, "href"> {
  name: string;
  entries: OSDownloadButtonEntry[];
}

export const OSDownloadButton = ({
  entries = [],
  name,
  ...props
}: OSDownloadButtonProps): JSX.Element | null => {
  const os = useOS();
  if (entries.length === 0) return null;
  let entry = entries.find((entry) => entry.os === os);
  if (entry == null) entry = entries[0];
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

const OSToUpdateFilePlatform: Record<OS, keyof UpdateFile["platforms"]> = {
  MacOS: "darwin-x86_64",
  Linux: "linux-x86_64",
  Windows: "windows-x86_64",
};

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/delta/release-spec.json";

export const DeltaDownloadButton = () => {
  console.log("Hello");
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);

  useAsyncEffect(async () => {
    const response = await fetch(JSON_URL);
    const updateFile = await response.json();
    console.log(updateFile);
    setUpdateFile(updateFile);
  }, []);

  console.log(updateFile);

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
