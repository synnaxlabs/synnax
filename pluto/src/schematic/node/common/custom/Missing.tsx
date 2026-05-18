// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/custom/missing.css";

import { NotFoundError, type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Flux } from "@/flux";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Symbol } from "@/schematic/symbol";
import { useRetrieveEffect } from "@/schematic/symbol/queries";
import { Text } from "@/text";

export type Resolution =
  | { status: "loading" }
  | { status: "missing" }
  | { status: "resolved"; spec: schematic.symbol.Spec };

/// useResolveSymbol drives the schematic symbol retrieve and reduces the Flux Result
/// to a three-state Resolution. A missing reference (NotFoundError) is treated as a
/// first-class state — the failure toast is suppressed so callers can render an
/// inline placeholder without spamming the user on every re-render.
export const useResolveSymbol = (specKey: string): Resolution => {
  const [resolution, setResolution] = useState<Resolution>({ status: "loading" });
  useRetrieveEffect({
    query: { key: specKey },
    addStatusOnFailure: false,
    onChange: useCallback((res: Flux.Result<schematic.symbol.Symbol>) => {
      if (res.variant === "success")
        setResolution({ status: "resolved", spec: res.data.data });
      else if (
        res.variant === "error" &&
        NotFoundError.matches(res.status.details.error)
      )
        setResolution({ status: "missing" });
    }, []),
  });
  return resolution;
};

export interface MissingProps {
  orientation?: location.Outer;
  className?: string;
  label?: string;
}

/// Missing renders a placeholder for a custom-symbol node whose underlying spec
/// cannot be resolved on the cluster — typically the symbol was deleted, or the
/// schematic was imported from a cluster that has not yet received the symbol.
/// The dashed outline + warning icon make the broken state legible on the canvas;
/// the property panel offers re-link or delete to recover.
export const Missing = ({
  orientation = "left",
  className,
  label,
}: MissingProps): ReactElement => (
  <div
    className={CSS(
      CSS.BM("symbol", "custom"),
      CSS.B("symbol-missing"),
      CSS.loc(orientation),
      className,
    )}
  >
    <Icon.Warning />
    {label != null && label.length > 0 && <Text.Text level="small">{label}</Text.Text>}
  </div>
);

/// MissingForm replaces the normal property-panel form for a custom-symbol node
/// whose referenced symbol is missing. Picking a replacement symbol writes the new
/// key into the node config; the existing stateOverrides are cleared because they
/// were keyed to the old spec's states and would not align with a different symbol.
export const MissingForm = ({ missingKey }: { missingKey: string }): ReactElement => {
  const form = Form.useContext();
  const handleRelink = useCallback(
    (key: schematic.symbol.Key | undefined) => {
      if (key == null) return;
      form.set("specKey", key);
      form.set("stateOverrides", []);
    },
    [form],
  );
  return (
    <Flex.Box y align="stretch" gap="small" className={CSS.B("symbol-missing-form")}>
      <Flex.Box x align="center" gap="small">
        <Icon.Warning color="var(--pluto-warning-z)" />
        <Text.Text level="p">
          The custom symbol referenced by this node is not available in this cluster.
        </Text.Text>
      </Flex.Box>
      <Text.Text level="small" color={8}>
        Symbol ID: {missingKey}
      </Text.Text>
      <Symbol.SelectSingle value={undefined} onChange={handleRelink} />
    </Flex.Box>
  );
};
