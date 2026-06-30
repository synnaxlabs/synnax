// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/modals/Footer.css";

import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/component/css";

export interface FooterProps extends Nav.BarProps {}

export const Footer = ({ className, ...rest }: FooterProps): ReactElement => (
  <Nav.Bar
    location="bottom"
    size="8rem"
    className={CSS(CSS.BE("modal", "footer"), className)}
    bordered
    {...rest}
  />
);
