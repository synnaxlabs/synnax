// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { staticTelem } from "@/telem/aether/static";
import { Canvas } from "@/vis/canvas";
import { Table } from "@/vis/table";

const story: Meta<typeof Table.Table> = {
  title: "Table",
  component: Table.Table,
};

const Example = (): ReactElement => {
  const telem = staticTelem.fixedString("One");
  const telem2 = staticTelem.fixedString("Two");

  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <Table.Table numColumns={2}>
        <Table.TR>
          <th>Name</th>
          <th>Value</th>
        </Table.TR>
        <Table.TR>
          <Table.StringTD telem={telem} />
          <Table.StringTD telem={telem2} />
        </Table.TR>
        <Table.TR>
          <Table.StringTD telem={telem} />
          <Table.StringTD telem={telem2} />
        </Table.TR>
      </Table.Table>
    </Canvas.Canvas>
  );
};

export const Default = () => <Example />;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export default story;
