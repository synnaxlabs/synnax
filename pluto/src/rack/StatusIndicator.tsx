// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/rack/StatusIndicator.css";

import { type rack } from "@synnaxlabs/client";
import { useEffect, useRef } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Status } from "@/status/core";
import { Tooltip } from "@/tooltip";

export interface StatusIndicatorProps {
  status?: rack.Status;
  tooltipLocation?: Tooltip.DialogProps["location"];
}

export const StatusIndicator = ({
  status,
  tooltipLocation = "right",
}: StatusIndicatorProps) => {
  const heartRef = useRef<SVGSVGElement>(null);
  const variant = status?.variant ?? "disabled";
  useEffect(() => {
    if (variant !== "success") return;
    const heart = heartRef.current;
    if (!heart) return;
    heart.classList.remove(CSS.BEM("rack", "heartbeat", "beat"));
    requestAnimationFrame(() =>
      heart.classList.add(CSS.BEM("rack", "heartbeat", "beat")),
    );
  }, [status]);
  return (
    <Tooltip.Dialog location={tooltipLocation}>
      <Status.Summary variant={variant} hideIcon level="small" weight={450}>
        {status?.message}
      </Status.Summary>
      <Icon.Heart
        ref={heartRef}
        className={CSS.BE("rack", "heartbeat")}
        style={{ color: Status.VARIANT_COLORS[variant] }}
      />
    </Tooltip.Dialog>
  );
};
