// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { CSSProperties } from "react";

export const ACCENT = "#5E94EE";
export const ERROR_ACCENT = "#EF4444";
export const WARNING_ACCENT = "#F59E0B";
export const LINE_IDLE = "rgba(255,255,255,0.05)";
export const TEXT_ON = "rgba(255,255,255,0.9)";
export const TEXT_OFF = "rgba(255,255,255,0.22)";
export const RULE_OFF = "rgba(255,255,255,0.06)";
export const VALUE_OFF = "rgba(255,255,255,0.12)";

export const LABEL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-code-font-family)",
  fontSize: "10px",
  fontWeight: 500,
  transition: "fill 0.5s ease",
};

export const VALUE_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-code-font-family)",
  fontSize: "11px",
  fontWeight: 400,
  transition: "fill 0.5s ease",
};

export const COMPUTE_LABEL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-code-font-family)",
  fontSize: "10px",
  fontWeight: 500,
  letterSpacing: "1px",
  transition: "fill 0.5s ease",
};

export const COMPUTE_DETAIL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-code-font-family)",
  fontSize: "8px",
  transition: "fill 0.5s ease",
};
