// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo as BaseLogo } from "./Logo";
import { LogoWatermark } from "./LogoWatermark";

export { type LogoProps } from "./Logo";

type BaseLogoType = typeof BaseLogo;

export interface LogoType extends BaseLogoType {
  Watermark: typeof LogoWatermark;
}

export const Logo = BaseLogo as LogoType;

Logo.Watermark = LogoWatermark;
