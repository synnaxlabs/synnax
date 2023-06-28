// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { Meta, StoryFn } from "@storybook/react";
import { XY } from "@synnaxlabs/x";

import { ValuePIDElementProps, ValuePIDElementSpec } from "./ValuePIDElement";

import { Canvas } from "@/core";

const story: Meta<typeof ValuePIDElementSpec.Element> = {
  title: "Vis/PID/ValuePIDElement",
  component: ValuePIDElementSpec.Element,
};

const Example = (): ReactElement => {
  const [props, setProps] = useState<ValuePIDElementProps>(
    ValuePIDElementSpec.initialProps
  );

  return (
    <>
      <ValuePIDElementSpec.Element
        {...props}
        editable={false}
        selected={false}
        position={XY.ZERO}
        onChange={setProps}
      />
      ;
      <ValuePIDElementSpec.Form value={props} onChange={setProps} />
    </>
  );
};

export const Primary = (): ReactElement => (
  <Canvas style={{ position: "fixed", height: "100%", width: "100%", top: 0, left: 0 }}>
    <Example />;
  </Canvas>
);

// eslint-disable-next-line import/no-default-export
export default story;
