// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space } from "@synnaxlabs/pluto";

import { Logo, LogoProps } from "./Logo";

import "./LogoWatermark.css";

/**
 * LogoWatermark displays the Synnax logo as a watermark in the center of the
 * screen.
 *
 * @param props - The same props as Logo.
 */
export const LogoWatermark = (props: LogoProps): JSX.Element => (
  <Space.Centered>
    <Logo className="delta-logo-watermark" {...props} />
  </Space.Centered>
);
