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
import { type ReactElement, useEffect, useState } from "react";

const VERSION_URL =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/synnax/pkg/version/VERSION";

export const WindowsDownloadButton = (): ReactElement => {
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
      size="large"
    >
      Download Synnax v{version} for Windows
    </Button.Link>
  );
};
