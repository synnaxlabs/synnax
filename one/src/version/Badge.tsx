// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/version/Badge.css";

import { Button } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { infoLayout } from "@/version/Info";
import { useSelect } from "@/version/selectors";
import { useCheckForUpdates } from "@/version/Updater";

export const Badge = (): ReactElement => {
  const v = useSelect();
  const placer = Layout.usePlacer();
  const updateAvailable = useCheckForUpdates();
  return (
    <Button.Button
      onClick={() => placer(infoLayout)}
      variant="text"
      size="medium"
      color={updateAvailable ? "var(--pluto-secondary-z)" : "var(--pluto-text-color)"}
    >
      {"v" + v}
    </Button.Button>
  );
};
