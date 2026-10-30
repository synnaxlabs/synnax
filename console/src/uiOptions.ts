// Scratch only: never commit. Turns on the modal proposal and points the dev Console at
// the private Core.

export const mountUIOptions = (): void => {
  try {
    if (localStorage.getItem("synnax-dev-connection-port") == null)
      localStorage.setItem("synnax-dev-connection-port", "9196");
  } catch {}
  document.documentElement.dataset.uiModal = "n";
  document.documentElement.dataset.uiTable = "ruled";
};
