// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/Logo/LogoWatermark.css";

import { type ReactElement } from "react";

import { Logo, type LogoProps } from "@/Logo/Logo";

/**
 * LogoWatermark displays the Synnax logo as a watermark in the center of the screen.
 *
 * @param props - The same props as Logo.
 */
export const LogoWatermark = (props: LogoProps): ReactElement => (
  <div className="synnax-logo-watermark__container">
    <Logo className="synnax-logo-watermark" {...props} />
  </div>
);
