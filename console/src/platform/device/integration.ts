// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";
import { type record } from "@synnaxlabs/x";
import { type ComponentType } from "react";

import { type ConnectProps } from "@/platform/device/types";

export interface Integration {
  /** The name a rack lists in its integrations when its Driver runs this one. */
  key: string;
  /** The display name, such as "OPC UA". */
  name: string;
  /** What the user connects, such as "OPC UA server". */
  title: string;
  description: string;
  icon: Icon.ReactElement;
  /** The make of the devices this integration connects. */
  make: string;
  /** The type of the scan task each Driver runs for this integration. */
  scanType: string;
}

/** An integration the user connects to by address. */
export interface Network extends Integration {
  Connect: ComponentType<ConnectProps>;
}

/** An integration whose hardware the Driver finds by scanning. */
export interface Hardware extends Integration {
  /**
   * What to check when the Driver does not find the hardware. The find view lists these
   * after a check that the Driver runs on the computer the hardware connects to.
   */
  checks: string[];
  /** The properties a device of this make starts with when the user configures it. */
  zeroProperties: record.Unknown;
}
