// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Base, type BaseProps } from "@/arc/handle/Base";
import { CSS } from "@/css";

export interface SourceProps extends Omit<BaseProps, "type"> {}

export const Source = ({ location, ...props }: SourceProps) => (
  <Base type="source" className={CSS.M("source")} location={location} {...props} />
);
