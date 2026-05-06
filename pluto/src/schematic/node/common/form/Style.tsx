// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@synnaxlabs/charon";
import { Form as Base } from "@/form";
import { Custom } from "@/schematic/node/common/custom";
import { ColorField } from "@/schematic/node/common/form/Color";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { Wrapper } from "@/schematic/node/common/form/Wrapper";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps } from "@/schematic/node/spec";

interface StyleFormProps extends FormProps {
  omit?: string[];
  hideInnerOrientation?: boolean;
  hideOuterOrientation?: boolean;
  showStateOverrides?: boolean;
}

export const StyleForm = ({
  omit,
  hideInnerOrientation,
  hideOuterOrientation,
}: StyleFormProps): ReactElement => {
  const hasStateOverrides =
    Base.useFieldValue<string>("stateOverrides", {
      optional: true,
    }) != null;
  return (
    <Wrapper x align="stretch" empty>
      <Flex.Box y grow>
        <Label.Form omit={omit} path="label" />
        <Flex.Box x grow>
          {!hasStateOverrides && <ColorField path="color" optional />}
          <Base.SwitchField
            path="normallyOpen"
            label="Normally Open"
            padHelpText={false}
            hideIfNull
            optional
          />
          <ScaleField path="scale" />
        </Flex.Box>
      </Flex.Box>
      {hasStateOverrides && <Custom.StateOverrideForm />}
      <Orientation.Field
        path=""
        hideInner={hideInnerOrientation}
        hideOuter={hideOuterOrientation}
      />
    </Wrapper>
  );
};
