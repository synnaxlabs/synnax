export enum Runtime {
  Browser = "browser",
  Node = "node",
}

const detectRuntime = (): Runtime => {
  if (
    typeof process !== "undefined" &&
    process.versions != null &&
    process.versions.node != null
  ) {
    return Runtime.Node;
  }

  if (typeof window === "undefined" || typeof window.document === "undefined") {
    console.warn("Freighter unable to safely detect runtime, assuming browser");
  }

  return Runtime.Browser;
};

export const RUNTIME = detectRuntime();
