import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Hardware } from "@/hardware";
import { Notifications as Core } from "@/notifications";
import { Version } from "@/version";

const NOTIFICATION_ADAPTERS: Core.Adapter[] = [
  ...Cluster.NOTIFICATION_ADAPTERS,
  ...Hardware.NOTIFICATION_ADAPTERS,
  ...Version.NOTIFICATION_ADAPTERS,
];

export const Notifications = (): ReactElement => (
  <Core.Notifications adapters={NOTIFICATION_ADAPTERS} />
);
