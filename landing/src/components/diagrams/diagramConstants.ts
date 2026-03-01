import type { CSSProperties } from "react";

export const ACCENT = "#5E94EE";
export const ERROR_ACCENT = "#EF4444";
export const WARNING_ACCENT = "#F59E0B";
export const LINE_IDLE = "rgba(255,255,255,0.05)";
export const TEXT_ON = "rgba(255,255,255,0.9)";
export const TEXT_OFF = "rgba(255,255,255,0.22)";
export const RULE_ON = ACCENT;
export const RULE_OFF = "rgba(255,255,255,0.06)";
export const VALUE_ON = "rgba(255,255,255,0.6)";
export const VALUE_OFF = "rgba(255,255,255,0.12)";

export const LABEL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-mono-font-family)",
  fontSize: "10px",
  fontWeight: 500,
  transition: "fill 0.5s ease",
};

export const VALUE_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-mono-font-family)",
  fontSize: "11px",
  fontWeight: 400,
  transition: "fill 0.5s ease",
};

export const COMPUTE_LABEL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-mono-font-family)",
  fontSize: "10px",
  fontWeight: 500,
  letterSpacing: "1px",
  transition: "fill 0.5s ease",
};

export const COMPUTE_DETAIL_STYLE: CSSProperties = {
  fontFamily: "var(--pluto-mono-font-family)",
  fontSize: "8px",
  transition: "fill 0.5s ease",
};
