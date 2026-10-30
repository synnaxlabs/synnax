// Scratch only: never commit. A floating bar that switches the open previews. Locked
// choices are constants here, so the bar shows only open questions.

import "@/uiPortMap.css";

import { useSyncExternalStore } from "react";
import { createRoot } from "react-dom/client";

type Option = "today" | "icons" | "word";

// Locked: icon and word.
export const useTaskSettingsOption = (): Option => "word";

export const useFooterTaskSettings = (): boolean => useTaskSettingsOption() !== "today";

type PortOption = "today" | "list" | "top" | "details" | "views";

// Locked (N1): list and ports.
export const usePortMapOption = (): PortOption => "views";

type HeaderOption = "today" | "name";

// Locked: name first.
export const useDetailsHeaderOption = (): HeaderOption => "name";

// Locked (N3): ports colored by state. The port map styles read this attribute.
document.documentElement.dataset.niPortColor = "state";

type DisabledOption = "today" | "outline" | "striped" | "faded";

const DISABLED_KEY = "ui-ni-disabled-option";

const readDisabled = (): DisabledOption => {
  try {
    const v = localStorage.getItem(DISABLED_KEY);
    if (v === "today" || v === "outline" || v === "faded") return v;
  } catch {}
  return "striped";
};

let currentDisabled = readDisabled();
const disabledListeners = new Set<() => void>();
document.documentElement.dataset.niDisabled = currentDisabled;

const setDisabled = (o: DisabledOption): void => {
  currentDisabled = o;
  document.documentElement.dataset.niDisabled = o;
  try {
    localStorage.setItem(DISABLED_KEY, o);
  } catch {}
  disabledListeners.forEach((l) => l());
};

const useDisabledOption = (): DisabledOption =>
  useSyncExternalStore(
    (l) => {
      disabledListeners.add(l);
      return () => disabledListeners.delete(l);
    },
    () => currentDisabled,
  );

const DISABLED_OPTIONS: Array<[DisabledOption, string]> = [
  ["today", "Today"],
  ["outline", "Outline"],
  ["striped", "Striped"],
  ["faded", "Faded"],
];

const buttonStyle = (active: boolean) => ({
  border: "none",
  borderRadius: 6,
  padding: "5px 8px",
  whiteSpace: "nowrap" as const,
  cursor: "pointer",
  font: "inherit",
  color: active ? "#fff" : "#bbb",
  background: active ? "#3e63dd" : "transparent",
});

const Bar = () => {
  const disabledStyle = useDisabledOption();
  return (
    <div
      style={{
        position: "fixed",
        bottom: 12,
        left: "50%",
        transform: "translateX(-50%)",
        zIndex: 100000,
        display: "flex",
        alignItems: "center",
        gap: 4,
        padding: 4,
        borderRadius: 9,
        background: "#1b1b1f",
        boxShadow: "0 4px 16px rgba(0,0,0,0.35)",
        font: "500 12px Inter, system-ui, sans-serif",
        color: "#ddd",
      }}
    >
      <span style={{ padding: "0 8px", color: "#999" }}>Disabled</span>
      {DISABLED_OPTIONS.map(([key, name]) => (
        <button
          key={key}
          onClick={() => setDisabled(key)}
          style={buttonStyle(disabledStyle === key)}
        >
          {name}
        </button>
      ))}
    </div>
  );
};

export const mountUIOptionsBar = (): void => {
  try {
    // The preview Core: clean, with only the NI test devices.
    localStorage.setItem("synnax-dev-connection-port", "9214");
  } catch {}
  const el = document.createElement("div");
  document.body.appendChild(el);
  createRoot(el).render(<Bar />);
};
