// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Embedded } from "@/cluster/embedded";
import { Remote } from "@/cluster/remote";
import { getFlags } from "@/getFlags";

export const Toolbar = (): ReactElement => {
  if (getFlags().community) return <Embedded.Toolbar />;
  return <Remote.Toolbar />;
};
