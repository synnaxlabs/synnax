// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import Synnax, { type SynnaxProps } from "@/client";

export const HOST = "localhost";
export const PORT = 9090;
const USERNAME = "synnax";
const PASSWORD = "seldon";

export const newClient = (...props: SynnaxProps[]): Synnax => {
  let _props = {};
  if (props.length > 0) _props = props[0];
  return new Synnax({
    host: HOST,
    port: PORT,
    username: USERNAME,
    password: PASSWORD,
    ..._props,
  });
};
