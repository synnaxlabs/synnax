import { useEffect, useState } from "react";

export const OperatingSystems = ["MacOS", "Windows", "Linux"] as const;
export type OS = typeof OperatingSystems[number];

export const useOS = (): OS | null => {
  const [os, setOS] = useState<OS | null>(null);
  useEffect(() => {
    console.log("EH");
    const userAgent = window.navigator.userAgent.toLowerCase();
    console.log(userAgent);
    if (userAgent.includes("mac")) {
      setOS("MacOS");
    } else if (userAgent.includes("win")) {
      setOS("Windows");
    } else if (userAgent.includes("linux")) {
      setOS("Linux");
    }
  }, []);
  return os;
};
