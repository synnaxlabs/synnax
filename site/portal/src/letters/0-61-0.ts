// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Letter } from "@/server/letter";

export const LETTER: Letter = {
  version: "0.61.0",
  subject: "Synnax v0.61: Arc modules, EtherCAT clock sync, MQTT, and Kafka",
  title: "Arc modules, EtherCAT clock sync, MQTT, Kafka, and updates you can undo",
  hero: "releases/0-61-0/hero",
  opening: [
    "Synnax v0.61 is out. Here are the four changes that I am most excited about.",
    "Live plots also draw in half the time, and a Core update is now safe to undo.",
  ],
  features: [
    {
      area: "Arc",
      title: "Arc modules",
      text:
        "Write a conversion, a check, or a sequence one time in a module, and import " +
        "it into every automation. A running automation keeps the module code that " +
        "it started with.",
      href: "/reference/control/arc/modules",
      image: "releases/0-61-0/arc-modules",
    },
    {
      area: "EtherCAT",
      title: "EtherCAT in clock sync",
      text:
        "Pick the distributed clock mode of each slave in the Console. The bus cycle " +
        "locks to the reference clock, so frames stay on time.",
      href: "/reference/hardware/ethercat/configure-device",
      image: "releases/0-61-0/ethercat-sync",
    },
    {
      area: "MQTT and Kafka",
      title: "MQTT, Sparkplug B, and Kafka",
      text:
        "Connect an MQTT broker or a Kafka cluster as a device, and read and write " +
        "channels with no Driver. Values that an edge node stored while offline land " +
        "at their own times.",
      href: "/reference/hardware/mqtt/connect-broker",
      image: "releases/0-61-0/mqtt",
    },
    {
      area: "Logs",
      title: "Logs that reach every stored value",
      text:
        "A log shows each value at its sample time and scrolls back through all of " +
        "your data. On a remote Core, the first lines show in 0.6 seconds instead " +
        "of 1.5.",
      href: "/reference/visualize/logs",
      image: "releases/0-61-0/logs",
    },
  ],
  breaking:
    "Arc syntax and some client calls changed. Read the breaking changes in the " +
    "release notes before you upgrade.",
  closing: [
    "Reply to this email with questions or feedback. Your reply comes straight to me.",
    "Emiliano",
  ],
};
