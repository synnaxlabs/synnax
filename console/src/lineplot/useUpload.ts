// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { type Flux, LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelectPendingUpload } from "@/lineplot/selectors";
import { clearPendingUpload } from "@/lineplot/slice";
import { Workspace } from "@/workspace";

// useAutoUpload sends the locally-staged pendingUpload to the server via
// Pluto's useCreate the first time the component mounts, then clears the
// flag. Returns true when no upload is pending (rendering may proceed).
export const useAutoUpload = (key: string): boolean => {
  const pendingUpload = useSelectPendingUpload(key);
  const name = Layout.useSelectRequiredName(key);
  const workspaceKey = Workspace.useSelectActiveKey();
  const dispatch = useDispatch();
  const { update: create } = PLinePlot.useCreate({
    afterSuccess: useCallback(
      ({ data: { key } }: Flux.AfterSuccessParams<PLinePlot.CreateOutput>) => {
        dispatch(clearPendingUpload({ key }));
      },
      [dispatch],
    ),
  });
  useEffect(() => {
    if (pendingUpload == null) return;
    create({
      key,
      name,
      workspace: workspaceKey ?? undefined,
      title: pendingUpload.title ?? lineplot.ZERO_NEW.title,
      legend: pendingUpload.legend ?? lineplot.ZERO_NEW.legend,
      channels: pendingUpload.channels ?? lineplot.ZERO_NEW.channels,
      ranges: pendingUpload.ranges ?? lineplot.ZERO_NEW.ranges,
      axes: pendingUpload.axes ?? lineplot.ZERO_NEW.axes,
      lines: pendingUpload.lines ?? lineplot.ZERO_NEW.lines,
      rules: pendingUpload.rules ?? lineplot.ZERO_NEW.rules,
    });
  }, [pendingUpload, workspaceKey, key, create, name]);
  return pendingUpload == null;
};
