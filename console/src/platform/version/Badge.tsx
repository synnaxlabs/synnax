// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Badge.css";

import { Button, Icon, Synnax, Tooltip } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useCheckForUpdates } from "@/platform/version/Updater";
import { useInfoModal } from "@/platform/version/useInfoModal";
import { Session } from "@/session";

/**
 * Version call-to-action for the nav bar. Renders nothing while the Console is
 * current and compatible; shows an update prompt when a new version is available
 * and a mismatch warning when the connected Core's version is incompatible.
 */
export const Badge = (): ReactElement | null => {
  const openInfo = useInfoModal();
  const updateAvailable = useCheckForUpdates();
  const connection = Synnax.useConnectionStatus();
  const mismatch =
    connection.variant === "success" && !connection.details.clientServerCompatible;
  if (!updateAvailable && !mismatch) return null;
  const { nodeVersion } = connection.details;
  return (
    <Tooltip.Dialog location={location.BOTTOM_LEFT}>
      {updateAvailable
        ? "A new version of the Console is ready to install"
        : `Core ${nodeVersion != null ? `v${nodeVersion} ` : ""}is incompatible ` +
          "with this version of the Console"}
      <Button.Button
        className={CSS.B("version-badge")}
        onClick={() => openInfo()}
        preventClick={Session.Runtime.ENGINE !== "tauri"}
        variant="text"
        size="small"
        level="small"
        textColor={
          updateAvailable ? "var(--pluto-secondary-z)" : "var(--pluto-warning-z)"
        }
        weight={500}
      >
        {updateAvailable ? <Icon.Download /> : <Icon.Warning />}
        {updateAvailable ? "Update Available" : "Version Mismatch"}
      </Button.Button>
    </Tooltip.Dialog>
  );
};
