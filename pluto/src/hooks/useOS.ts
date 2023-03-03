import { useEffect, useState } from "react";

import { OS } from "@synnaxlabs/x";

export const useOS = (force?: OS, default_: OS | null = null): OS | null =>
  getOS(force, default_);

export const getOS = (force?: OS, default_: OS | null = null): OS | null => {
  if (force != null) return force;
  if (typeof window === "undefined") return null;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) {
    return "MacOS";
  } else if (userAgent.includes("win")) {
    return "Windows";
  } else if (userAgent.includes("linux")) {
    return "Linux";
  }
  return default_;
};
