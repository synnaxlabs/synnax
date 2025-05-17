import { Symbol } from "@/vis/slate/symbols/notification/Notification";

export const REGISTRY = {
  notification: {
    key: "notification",
    name: "Notification",
    zIndex: 100,
    Form: () => null,
    Symbol,
    defaultProps: () => ({}),
    Preview: () => null,
  },
};
