// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Core, type CoreProps } from "@/arc/handle/Core";
import { CSS } from "@/css";

export interface SinkProps extends Omit<CoreProps, "type"> {}

export const Sink = ({ location, ...props }: SinkProps) => (
  <Core type="target" className={CSS.M("sink")} location={location} {...props} />
);
