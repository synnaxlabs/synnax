// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/version/Badge.css";

import { Button } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { INFO_LAYOUT } from "@/version/Info";
import { useSelectVersion } from "@/version/selectors";
import { useCheckForUpdates } from "@/version/Updater";

export const Badge = () => {
  const version = useSelectVersion();
  const placeLayout = Layout.usePlacer();
  const updateAvailable = useCheckForUpdates();
  return (
    <Button.Button
      onClick={() => placeLayout(INFO_LAYOUT)}
      variant="text"
      size="medium"
      color={updateAvailable ? "var(--pluto-secondary-z)" : "var(--pluto-text-color)"}
    >
      {`v${version}`}
    </Button.Button>
  );
};
