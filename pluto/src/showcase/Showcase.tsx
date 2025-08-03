// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { state } from "@/state";

import { BreadcrumbShowcase } from "./BreadcrumbShowcase";
import { ButtonShowcase } from "./ButtonShowcase";
import { DISPLAY, PADDING_STYLE } from "./constants";
import { DisplaySelector } from "./DisplaySelector";
import { DividerShowcase } from "./DividerShowcase";
import { FlexShowcase } from "./FlexShowcase";
import { InputShowcase } from "./InputShowcase";
import { SelectShowcase } from "./SelectShowcase";
import { TagShowcase } from "./TagShowcase";
import { TextShowcase } from "./TextShowcase";

export const Showcase = () => {
  const [display, setDisplay] = state.usePersisted<(typeof DISPLAY)[number][]>(
    DISPLAY,
    "display",
  );
  return (
    <Flex.Box y gap="large" style={PADDING_STYLE}>
      <DisplaySelector display={display} setDisplay={setDisplay} />
      <Flex.Box x>
        {display.includes("text") && <TextShowcase />}
        {display.includes("button") && <ButtonShowcase />}
        {display.includes("tag") && <TagShowcase />}
      </Flex.Box>
      <Flex.Box x>{display.includes("input") && <InputShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("select") && <SelectShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("flex") && <FlexShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("breadcrumb") && <BreadcrumbShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("divider") && <DividerShowcase />}</Flex.Box>
    </Flex.Box>
  );
};
