// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo as CoreLogo } from "./Logo";
import { LogoWatermark } from "./LogoWatermark";

export { type LogoProps } from "./Logo";

type CoreLogoType = typeof CoreLogo;

export interface LogoType extends CoreLogoType {
  Watermark: typeof LogoWatermark;
}

export const Logo = CoreLogo as LogoType;

Logo.Watermark = LogoWatermark;
