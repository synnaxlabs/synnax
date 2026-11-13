// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";
import { z } from "zod";

import { useDisableFilter } from "@/feature/ethercat/device/queries";
import { type Device } from "@/platform/device";
import { Modals } from "@/platform/modals";

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
  const { update, variant } = useDisableFilter();
  const filtered = scanStatusDataZ.parse(scanner.status?.details.data)?.filtered ?? [];
  if (filtered.length === 0) return null;
  const handleClick = () =>
    handleError(async () => {
      const confirmed = await confirm({
        message: "Turn off the DewesoftX filter?",
        description:
          "DewesoftX cannot use the adapter while its filter is off. Windows can ask " +
          "for administrator access on the computer that runs the Driver.",
        confirm: { label: "Turn off", variant: "warning" },
      });
      if (confirmed === true) update({ scanner });
    }, "Failed to turn off the DewesoftX filter");
  return (
    <Button.Button
      size="small"
      variant="outlined"
      status={status.keepVariants(variant, "loading")}
      onClick={handleClick}
    >
      Turn off the filter
    </Button.Button>
  );
};
