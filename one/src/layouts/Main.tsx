// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Main.css";

import { Drift } from "@synnaxlabs/drift";
import { Align, Haul, Text } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { getCurrentWebview } from "@tauri-apps/api/webview";
import { CSSProperties, type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { NavDrawer } from "@/components/nav/Nav";
import { Device } from "@/hardware/device";
import { Layout } from "@/layout";
import { Mosaic } from "@/layouts/Mosaic";
import { NavBottom, NavLeft, NavRight, NavTop } from "@/layouts/Nav";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { Notifications } from "@/notifications";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const NOTIFICATION_ADAPTERS = [
  ...Device.NOTIFICATION_ADAPTERS,
  ...Version.NOTIFICATION_ADAPTERS,
  ...Cluster.NOTIFICATION_ADAPTERS,
];

const LINK_HANDLERS: Link.Handler[] = [
  ChannelServices.linkHandler,
  LinePlotServices.linkHandler,
  RangeServices.linkHandler,
  SchematicServices.linkHandler,
  Workspace.linkHandler,
];

const SideEffect = (): null => {
  const dispatch = useDispatch();
  useEffect(() => {
    dispatch(Layout.maybeCreateGetStartedTab());
  }, []);
  Version.useLoadTauri();
  Device.useListenForChanges();
  Workspace.useSyncLayout();
  Link.useDeep({ handlers: LINK_HANDLERS });
  Layout.useTriggers();
  const dragging = Haul.useDraggingState();
  return null;
};

export const MAIN_TYPE = Drift.MAIN_WINDOW;

const FileDrop = () => {
  const canDrop = Haul.canDropOfType(Haul.FILE_TYPE);

  const handleDrop = useMutation({
    mutationKey: ["file", "drop"],
    mutationFn: async (props: Haul.OnDropProps) => {
      // get the file path from the event
      console.log(props);
      if (!props.event) return;
      const files = Array.from(props.event.dataTransfer.files);
      console.log(files);
      // await fetch("http://localhost:5000", files[0].
    },
  });

  const dropProps = Haul.useDrop({
    type: "file-x",
    onDrop: handleDrop.mutate,
    canDrop: () => true,
  });

  const dragging = Haul.useDraggingState();
  const isDragging = canDrop(dragging);

  const style: CSSProperties = {
    position: "fixed",
    top: 0,
    left: 0,
    height: 0,
    width: 0,
    zIndex: -1,
  };
  if (isDragging) {
    style.width = "100%";
    style.height = "100%";
    style.background = "var(--pluto-primary-z-30)";
    style.zIndex = 5000;
  }

  return (
    <Align.Space {...dropProps} style={style}>
      <Text.Text level="h4">Drop to upload</Text.Text>
    </Align.Space>
  );
};

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const Main = (): ReactElement => (
  <>
    {/* We need to place notifications here so they are in the proper stacking context */}
    <FileDrop />
    <Notifications.Notifications adapters={NOTIFICATION_ADAPTERS} />
    <SideEffect />
    <NavTop />
    <Layout.Modals />
    <Align.Space className="console-main-fixed--y" direction="x" empty>
      <NavLeft />
      <Align.Space
        className="console-main-content-drawers console-main-fixed--y console-main-fixed--x"
        empty
      >
        <Align.Space className="console-main--driven" direction="x" empty>
          <NavDrawer location="left" />
          <main className="console-main--driven" style={{ position: "relative" }}>
            <Mosaic />
          </main>
          <NavDrawer location="right" />
        </Align.Space>
        <NavDrawer location="bottom" />
      </Align.Space>
      <NavRight />
    </Align.Space>
    <NavBottom />
  </>
);
