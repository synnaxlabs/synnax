// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/AuthProvider";
import { pb } from "@/util/pocketbase";

const VERSION_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/core/pkg/version/VERSION";

const POCKETBASE_URL =
  import.meta.env.PUBLIC_POCKETBASE_URL ?? "https://api.synnaxlabs.com";

export const WindowsDownloadButton = (): ReactElement | null => {
  const { isAuthenticated, showAuthModal } = useAuth();
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    fetch(VERSION_URL)
      .then(async (r) => r.text())
      .then((v) => setVersion(v.trim()))
      .catch(() => {});
  }, []);

  const handleClick = useCallback(async () => {
    if (!isAuthenticated) {
      showAuthModal();
      return;
    }
    try {
      const res = await pb.send("/api/downloads/token", { method: "POST" });
      const url = `${POCKETBASE_URL}/api/downloads/core-installer/windows/${res.token}`;
      window.location.href = url;
    } catch {
      showAuthModal();
    }
  }, [isAuthenticated, showAuthModal]);

  if (version == null) return null;

  return (
    <Button.Button size="large" variant="filled" onClick={handleClick}>
      <Icon.Download />
      Download Synnax v{version} for Windows
    </Button.Button>
  );
};
