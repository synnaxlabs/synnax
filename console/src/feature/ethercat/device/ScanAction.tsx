// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { type Device } from "@/platform/device";
import { Modals } from "@/platform/modals";

/** The scan task command that turns off the DewesoftX filter. */
const DISABLE_DEWESOFT_FILTER_COMMAND = "disable_dewesoft_filter";

/** The Driver waits up to 45 seconds for the user to approve the change in Windows. */
const DISABLE_TIMEOUT = TimeSpan.seconds(60);

const scanStatusDataZ = z
  .object({ filtered: z.array(z.string()).default(() => []) })
  .optional();

/**
 * Turns off the DewesoftX filter on the adapters whose replies the last scan found it
 * blocks, once the user confirms. Renders nothing when the filter blocks none.
 */
export const ScanAction = ({
  scanner,
}: Device.ScanActionProps): ReactElement | null => {
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  const [disabling, setDisabling] = useState(false);
  const filtered = scanStatusDataZ.parse(scanner.status?.details.data)?.filtered ?? [];
  if (filtered.length === 0) return null;
  const handleClick = () =>
    handleError(async () => {
      const confirmed = await confirm({
        message: "Turn off the DewesoftX filter?",
        description:
          "DewesoftX cannot use the adapter while its filter is off. Windows can ask " +
          "for administrator access.",
        confirm: { label: "Turn off", variant: "warning" },
      });
      if (confirmed !== true) return;
      setDisabling(true);
      await scanner
        .executeCommandSync({
          type: DISABLE_DEWESOFT_FILTER_COMMAND,
          timeout: DISABLE_TIMEOUT,
        })
        .finally(() => setDisabling(false));
    }, "Failed to turn off the DewesoftX filter");
  return (
    <Button.Button
      size="small"
      variant="outlined"
      status={disabling ? "loading" : undefined}
      onClick={handleClick}
    >
      Turn off the filter
    </Button.Button>
  );
};
