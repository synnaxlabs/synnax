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
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/AuthProvider";
import { pb } from "@/util/pocketbase";

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

const POCKETBASE_URL =
  import.meta.env.PUBLIC_POCKETBASE_URL ?? "https://api.synnaxlabs.com";

export const DownloadButton = (): ReactElement | null => {
  const { isAuthenticated, showAuthModal } = useAuth();
  const [updateFile, setUpdateFile] = useState<UpdateFile | null>(null);

  useEffect(() => {
    fetch(JSON_URL)
      .then(async (response) => (await response.json()) as UpdateFile)
      .then(setUpdateFile)
      .catch(() => setUpdateFile(null));
  }, []);

  const handleClick = useCallback(async () => {
    if (!isAuthenticated) {
      showAuthModal();
      return;
    }
    if (updateFile == null) return;
    try {
      const os = runtime.getOS();
      const suffix = SUFFIXES[os];
      if (suffix == null) return;
      const res = await pb.send("/api/downloads/token", { method: "POST" });
      const url = `${POCKETBASE_URL}/api/downloads/console/${os.toLowerCase()}/${res.token}`;
      window.location.href = url;
    } catch {
      showAuthModal();
    }
  }, [isAuthenticated, showAuthModal, updateFile]);

  if (updateFile == null) return null;
  const version = updateFile.version;
  const os = runtime.getOS();
  const suffix = SUFFIXES[os];
  if (suffix == null) return null;

  return (
    <Button.Button variant="filled" onClick={handleClick}>
      <Icon.Download />
      Download {version} for {os}
    </Button.Button>
  );
};
