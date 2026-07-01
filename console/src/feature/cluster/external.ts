import { versionOutdatedAdapter } from "@/feature/cluster/notification";
import { type Notifications } from "@/platform/notifications";

export * from "@/feature/cluster/ConnectionBadge";
export * from "@/feature/cluster/link";
export * from "@/feature/cluster/palette";

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  versionOutdatedAdapter,
];
