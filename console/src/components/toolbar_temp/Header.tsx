// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/toolbar/Header.css";

import { Header as PHeader } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

export interface HeaderProps extends Omit<PHeader.HeaderProps, "level" | "divided"> {}

export const Header = (props: HeaderProps) => (
  <PHeader.Header
    className={CSS.B("toolbar-header")}
    level="h5"
    shrink={false}
    {...props}
  />
);
