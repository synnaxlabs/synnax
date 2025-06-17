// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import Synnax, { type SynnaxProps } from "@/client";

export const DEFAULT_PROPS: SynnaxProps = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
};

export const newTestClient = (props?: Partial<SynnaxProps>): Synnax =>
  new Synnax({ ...DEFAULT_PROPS, ...props });
