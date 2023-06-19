// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Meta, StoryFn } from "@storybook/react";

import { VisCanvas } from "../Canvas";

import { Valve } from "./main";

import { StaticTelem } from "@/telem/static/main";

const story: Meta<typeof Valve> = {
  title: "Vis/Valve",
  component: Valve,
};

const Example = (): ReactElement => {
  const telem = StaticTelem.usePoint(1);

  return (
    <VisCanvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <Valve color="#fc3d03" />
    </VisCanvas>
  );
};

export const Primary: StoryFn<typeof Value> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
