type Runtime = "browser" | "node";

const detectRuntime = (): Runtime => {
  if (
    typeof process !== "undefined" &&
    process.versions != null &&
    process.versions.node != null
  )
    return "node";

  if (window === undefined || window.document === undefined)
    console.warn("freighter unable to safely detect runtime, assuming browser");

  return "browser";
};

export const RUNTIME = detectRuntime();
