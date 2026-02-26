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
  const tokenURL = `${POCKETBASE_URL}/api/downloads/driver/${platform}/${token}`;

  if (os === "ni-linux-rt") {
    return (
      <code className="inline-code">
        curl -LO {tokenURL} && chmod +x install-driver-nilinuxrt.sh &&
        ./install-driver-nilinuxrt.sh
      </code>
    );
  }

  if (os === "windows") {
    return (
      <code className="inline-code">
        Invoke-WebRequest -Uri &quot;{tokenURL}&quot; -OutFile
        &quot;synnax-driver.exe&quot;
      </code>
    );
  }

  return <code className="inline-code">curl -LO {tokenURL}</code>;
};
