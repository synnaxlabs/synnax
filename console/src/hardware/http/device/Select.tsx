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

import { EmptyAction } from "@/components";
import { Common } from "@/hardware/common";
import { CONNECT_LAYOUT } from "@/hardware/http/device/Connect";
import { MAKE } from "@/hardware/http/device/types";
import { Layout } from "@/layout";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  const handleClick = useCallback(() => {
    placeLayout(CONNECT_LAYOUT);
    close();
  }, [placeLayout, close]);
  return (
    <EmptyAction
      message="No HTTP servers connected."
      action="Connect a new server"
      onClick={handleClick}
    />
  );
};

const emptyContent = <EmptyContent />;

export const Select = () => (
  <Common.Device.Select
    configureLayout={CONNECT_LAYOUT}
    emptyContent={emptyContent}
    label="HTTP Server"
    make={MAKE}
  />
);
