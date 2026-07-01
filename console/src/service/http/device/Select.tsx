// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Empty } from "@/component/empty";
import { Device } from "@/component/device";
import { MAKE } from "@/service/http/device/types";
import { useConnectModal } from "@/service/http/device/useConnectModal";

const EmptyContent = () => {
  const connect = useConnectModal();
  const { close } = Dialog.useContext();
  const handleClick = useCallback(() => {
    connect();
    close();
  }, [connect, close]);
  return (
    <Empty.Action
      message="No HTTP servers connected."
      action="Connect a new server"
      onClick={handleClick}
    />
  );
};

const emptyContent = <EmptyContent />;

export const Select = () => {
  const connect = useConnectModal();
  return (
    <Device.Select
      onConfigure={(deviceKey) => connect({ deviceKey })}
      emptyContent={emptyContent}
      label="HTTP server"
      make={MAKE}
    />
  );
};
