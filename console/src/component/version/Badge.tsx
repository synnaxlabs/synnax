// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/version/Badge.css";

import { Button } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useCheckForUpdates } from "@/component/version/Updater";
import { useInfoModal } from "@/component/version/useInfoModal";
import { Session } from "@/session";

export const Badge = (): ReactElement => {
  const version = Session.Version.use();
  const openInfo = useInfoModal();
  const updateAvailable = useCheckForUpdates();
  return (
    <Button.Button
      onClick={() => openInfo()}
      preventClick={Session.Runtime.ENGINE !== "tauri"}
      variant="text"
      size="small"
      level="small"
      textColor={updateAvailable ? "var(--pluto-secondary-z)" : 9}
      weight={500}
      contrast={2}
    >
      {`v${version}`}
    </Button.Button>
  );
};
