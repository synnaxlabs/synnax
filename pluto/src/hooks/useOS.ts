import { useEffect, useState } from "react";

import { OS } from "@synnaxlabs/x";

export const useOS = (): OS | null => {
  const [os, setOS] = useState<OS | null>(null);
  useEffect(() => {
    const os = getOS();
    setOS(os);
  }, []);
  return os;
};

export const getOS = (): OS | null => {
  if (typeof window === "undefined") return null;
  const userAgent = window.navigator.userAgent.toLowerCase();
  if (userAgent.includes("mac")) {
    return "MacOS";
  } else if (userAgent.includes("win")) {
    return "Windows";
  } else if (userAgent.includes("linux")) {
    return "Linux";
  }
  return null;
};
