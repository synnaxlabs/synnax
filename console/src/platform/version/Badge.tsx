// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Badge.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useCheck } from "@/platform/version/Updater";
import { useInfoModal } from "@/platform/version/useInfoModal";

/**
 * Version call-to-action for the nav bar. Renders nothing while the Console is current,
 * and an update prompt when a new version is available.
 */
export const Badge = (): ReactElement | null => {
  const openInfo = useInfoModal();
  const check = useCheck();
  if (check.variant !== "success" || check.available == null) return null;
  return (
    <Tooltip.Dialog location={location.BOTTOM_LEFT}>
      A new version of the Console is ready to install
      <Button.Button
        className={CSS.B("version-badge")}
        onClick={() => openInfo()}
        variant="text"
        size="small"
        level="small"
        textColor="var(--pluto-secondary-z)"
        weight={500}
      >
        <Icon.Download />
        Update available
      </Button.Button>
    </Tooltip.Dialog>
  );
};
