// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { useConnectModal } from "@/feature/embedded/useConnectModal";

/** A nav bar button that opens the dialog that shows how to connect from code. */
export const ConnectButton = (): ReactElement => {
  const open = useConnectModal();
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={() => open()}
      tooltip="Connect from code"
    >
      <Icon.Code />
    </Button.Button>
  );
};
