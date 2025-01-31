// Copyright 2025 Synnax Labs, Inc.
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

interface OSDownloadButtonEntry {
  href: string;
  os: runtime.OS;
}

interface OSDownloadButtonProps extends Omit<Button.LinkProps, "href"> {
  entries: OSDownloadButtonEntry[];
  name: string;
}

const OSDownloadButton = ({
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

const JSON_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/console/release-spec.json";

export const SynnaxConsoleDownloadButton = (): ReactElement | null => {
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
  return (
    <OSDownloadButton
      name={version}
      size="large"
      entries={[
        { os: "macOS", href: `${baseURL}_aarch64.dmg` },
        { os: "Windows", href: `${baseURL}_x64-setup.exe` },
      ]}
    />
  );
};

const VERSION_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/synnax/pkg/version/VERSION";

export const SynnaxServerDownloadButton = (): ReactElement => {
  const [version, setVersion] = useState<string | null>(null);
  useEffect(() => {
    fetch(VERSION_URL)
      .then(async (response) => await response.text())
      .then(setVersion)
      .catch(console.error);
  }, []);
  return (
    <Button.Link
      href={`https://github.com/synnaxlabs/synnax/releases/download/synnax-v${version}/synnax-setup-v${version}.exe`}
      startIcon={<Icon.Download />}
      className="os-download-button"
      size="large"
    >
      Download v{version} Installer for Windows
    </Button.Link>
  );
};
