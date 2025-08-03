// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@/button";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

import { PADDING_STYLE, THIN_PADDING_STYLE } from "./constants";
import { InputShowcaseText } from "./InputShowcase";

const INPUT_PLACEHOLDER = (
  <>
    <Icon.Search />
    Catalyst
  </>
);

export const FlexShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1} full="x">
    <Flex.Box x>
      <Flex.Box y bordered rounded={1} style={THIN_PADDING_STYLE}>
        <Text.Text>Packed Pairs</Text.Text>
        <Flex.Box x pack rounded={2}>
          <Button.Button variant="filled">Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box y pack sharp>
          <Button.Button>Hello</Button.Button>
          <Button.Button variant="filled">Hello</Button.Button>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y bordered rounded={1} style={THIN_PADDING_STYLE}>
        <Text.Text>Packed Triples</Text.Text>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button variant="filled">Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box y pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button variant="filled">Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y bordered rounded={1} style={THIN_PADDING_STYLE}>
        <Text.Text>Packed Quadruples</Text.Text>
        <Flex.Box y pack>
          <Flex.Box x pack>
            <Button.Button variant="filled">Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box x pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button variant="filled">Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x pack>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button variant="filled">Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y pack>
            <Button.Button variant="filled">Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x bordered rounded={1} style={THIN_PADDING_STYLE}>
      <Text.Text>Packed Grid</Text.Text>
      <Flex.Box y pack>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button variant="filled">Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
      </Flex.Box>

      <Flex.Box y>
        <Flex.Box x pack>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box y bordered rounded={1} style={THIN_PADDING_STYLE}>
      <Text.Text>Nested Packs Same Direction</Text.Text>
      <Flex.Box y>
        <Flex.Box x pack>
          <Flex.Box x pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Flex.Box x pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
          <Flex.Box x pack>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box x>
        <Flex.Box y pack>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
          <Button.Button>Hello</Button.Button>
        </Flex.Box>
        <Flex.Box y pack>
          <Button.Button>Hello</Button.Button>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box y pack>
          <Button.Button>Hello</Button.Button>
          <Button.Button>Hello</Button.Button>
          <Flex.Box y pack>
            <Button.Button>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x bordered rounded={1} style={THIN_PADDING_STYLE}>
      <Text.Text>Nested Packs Input</Text.Text>
      <Flex.Box y>
        <Flex.Box pack>
          <InputShowcaseText placeholder={INPUT_PLACEHOLDER} />
          <Button.Button>
            <Icon.Search />
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box pack y>
        <InputShowcaseText placeholder="Search" />
        <Button.Button full="x" justify="center" variant="filled">
          <Icon.Search />
          Search
        </Button.Button>
      </Flex.Box>
      <Flex.Box pack y>
        <Button.Button full="x" justify="center" variant="filled">
          <Icon.Search />
          Search
        </Button.Button>
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} />
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
