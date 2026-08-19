// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Scale } from "@/schematic/node/common/scale";
import { DEFAULT_DIMENSIONS } from "@/schematic/node/general/scale/config";
import { Tabs } from "@/tabs";

export const ScaleForm = (): ReactElement => (
  <Tabs.Frame initialValue="properties">
    <Tabs.Selector>
      <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="properties">
      <Form.Wrapper x>
        <Flex.Box y grow>
          <Label.Form path="label" />
          <Flex.Box x>
            <Base.NumericField
              path="dimensions.width"
              label="Width"
              defaultValue={DEFAULT_DIMENSIONS.width}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
            <Base.NumericField
              path="dimensions.height"
              label="Height"
              defaultValue={DEFAULT_DIMENSIONS.height}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
            <Scale.Form path="indicator" />
          </Flex.Box>
        </Flex.Box>
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Form.Wrapper x>
        <Flex.Box y grow>
          <Scale.TelemForm path="indicator" />
        </Flex.Box>
      </Form.Wrapper>
    </Tabs.Content>
  </Tabs.Frame>
);
