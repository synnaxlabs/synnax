// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/AuthProvider";
import { pb } from "@/util/pocketbase";

const VERSION_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/core/pkg/version/VERSION";

const POCKETBASE_URL =
  import.meta.env.PUBLIC_POCKETBASE_URL ?? "https://api.synnaxlabs.com";

interface DownloadURLProps {
  os: string;
}

export const DownloadURL = ({ os }: DownloadURLProps): ReactElement => {
  const { isAuthenticated, showAuthModal } = useAuth();
  const [version, setVersion] = useState<string | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetch(VERSION_URL)
      .then(async (r) => r.text())
      .then((v) => setVersion(v.trim()))
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      setToken(null);
      return;
    }
    setLoading(true);
    pb.send("/api/downloads/token", { method: "POST" })
      .then((res: { token: string }) => setToken(res.token))
      .catch(() => setToken(null))
      .finally(() => setLoading(false));
  }, [isAuthenticated]);

  if (!isAuthenticated) {
    return (
      <Button.Button variant="outlined" onClick={showAuthModal}>
        Sign in to view download command
      </Button.Button>
    );
  }

  if (version == null || loading || token == null) {
    return <code className="inline-code">Loading...</code>;
  }

  const platform = os.toLowerCase();
  const tokenURL = `${POCKETBASE_URL}/api/downloads/core/${platform}/${token}`;

  if (os === "windows") {
    const code = `$ErrorActionPreference="Stop"; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $ProgressPreference='SilentlyContinue'; $null=New-Item -Type Directory -Force $env:appdata/synnax; Invoke-WebRequest -Uri ${tokenURL} -OutFile synnax.exe; Copy-Item -Force "synnax.exe" -Destination $env:appdata/synnax; if (-not [Environment]::GetEnvironmentVariable("Path", "User").Contains("%APPDATA%\\synnax")) { [Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";%APPDATA%\\synnax", "User"); }; $env:PATH += ";$env:appdata\\synnax"`;
    return <code className="inline-code">{code}</code>;
  }

  return <code className="inline-code">curl -LO {tokenURL}</code>;
};
