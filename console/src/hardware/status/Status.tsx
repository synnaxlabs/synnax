import { type ReactElement } from "react";

import { Align, Status as PStatus } from "@synnaxlabs/pluto";
import { Text } from "@synnaxlabs/pluto/text";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

import "@/hardware/status/Status.css";

export const Status: Layout.Renderer = (): ReactElement => {
  return (
    <Align.Space className={CSS.B("hardware-status")} direction="y" size={8}>
      <Align.Space className={CSS.B("header")} direction="x" justify="spaceBetween">
        <Align.Space className={CSS.B("title")} direction="x" align="end">
          <Text.Text level="h1">GSE DAQ</Text.Text>
          <Text.Text level="h3">Node 2 / Rack 2</Text.Text>
        </Align.Space>
        <Align.Space className={CSS.B("status")} direction="x" align="center">
          <PStatus.Text level="p" variant="success">
            Good Hearbeat
          </PStatus.Text>
          <PStatus.Text level="p" variant="success">
            All Modules Alive
          </PStatus.Text>
        </Align.Space>
      </Align.Space>
      <Align.Space className={CSS.B("modules")} empty>
        <Align.Space className={CSS.B("module")} direction="x" justify="spaceBetween">
          <Align.Space className="title" direction="y" empty>
            <Text.Text level="p">Analog In</Text.Text>
            <Text.Text level="p">NI Analog Reader</Text.Text>
          </Align.Space>
          <PStatus.Text level="p" variant="success">
            Valid Config
          </PStatus.Text>
          <PStatus.Text level="p" variant="success">
            Data Saving On
          </PStatus.Text>
          <PStatus.Text level="p" variant="success">
            Running
          </PStatus.Text>
        </Align.Space>
        <Align.Space className={CSS.B("module")} direction="x" justify="spaceBetween">
          <Align.Space className="title" direction="y" empty>
            <Text.Text level="p">Digital Out</Text.Text>
            <Text.Text level="p">NI Analog Reader</Text.Text>
          </Align.Space>
          <PStatus.Text level="p" variant="error">
            Config error
          </PStatus.Text>
          <PStatus.Text level="p" variant="warning">
            Data Saving Off
          </PStatus.Text>
          <PStatus.Text level="p" variant="error">
            Stopped
          </PStatus.Text>
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

export type LayoutType = "hardwareStatus";
export const LAYOUT_TYPE: LayoutType = "hardwareStatus";

export const create =
  (initial: Omit<Partial<Layout.LayoutState>, "type">): Layout.Creator =>
  () => {
    const { name = "Hardware Status", location = "mosaic", ...rest } = initial;
    return {
      key: initial.key ?? uuidv4(),
      type: LAYOUT_TYPE,
      name,
      location,
      ...rest,
    };
  };
