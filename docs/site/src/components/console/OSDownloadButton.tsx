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
    "darwin-x86_64": { url: string };
    "linux-x86_64": { url: string };
    "windows-x86_64": { url: string };
  };
}

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
          href: `https://github.com/synnaxlabs/synnax/releases/download/console-${updateFile.version}/Synnax_${updateFile.version.slice(1)}_x64-setup.exe`,
        },
      ]}
    />
  );
};

export const SynnaxServerDownloadButton = (): ReactElement => {
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);

  useEffect(() => {
    fetch(JSON_URL)
      .then(async (response) => await response.json())
      .then((f) => {
        console.log("Fetched update file:", f);
        setUpdateFile(f as UpdateFile);
      })
      .catch((error) => {
        console.error("Error fetching update file:", error);
        setUpdateFile(null);
      });
  }, []);

  if (updateFile == null) return <Button.Button disabled>Loading...</Button.Button>;

  const version = updateFile.version;

  return (
    <Button.Link
      href={`https://github.com/synnaxlabs/synnax/releases/download/synnax-${version}/synnax-setup-${version}.exe`}
      startIcon={<Icon.Download />}
      className="os-download-button"
      size="large"
    >
      Download Synnax-{version} for Windows
    </Button.Link>
  );
};
