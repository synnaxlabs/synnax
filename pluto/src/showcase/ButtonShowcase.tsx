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

import { PADDING_STYLE } from "./constants";

export const ButtonShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Flex.Box y>
        <Button.Button size="huge">Hello</Button.Button>
        <Button.Button size="large">Hello</Button.Button>
        <Button.Button size="medium">Hello</Button.Button>
        <Button.Button size="small">Hello</Button.Button>
        <Button.Button size="tiny">Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="large" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="small" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="large" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="small" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <Flex.Box y>
        <Button.Button size="huge">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Flex.Box x>
          <Button.Button disabled>Hello</Button.Button>
          <Button.Button disabled variant="filled">
            Hello
          </Button.Button>
          <Button.Button disabled variant="text">
            Hello
          </Button.Button>
        </Flex.Box>
        <Flex.Box x>
          <Button.Button sharp>Hello</Button.Button>
          <Button.Button status="loading">Hello</Button.Button>
          <Button.Button status="loading">
            <Icon.Auto />
          </Button.Button>
          <Button.Button href="https://www.google.com" variant="text">
            Link to Google
          </Button.Button>
        </Flex.Box>
        <Flex.Box x>
          <Button.Button color="#12E3E2">Hello</Button.Button>
          <Button.Button color="#12E3E2" variant="filled">
            Hello
          </Button.Button>
          <Button.Button color="#12E3E2" variant="text">
            Hello
          </Button.Button>
        </Flex.Box>
        <Flex.Box x>
          <Button.Button status="warning">Hello</Button.Button>
          <Button.Button status="warning" variant="filled">
            Hello
          </Button.Button>
          <Button.Button status="warning" variant="text">
            Hello
          </Button.Button>
        </Flex.Box>
        <Flex.Box x>
          <Button.Button status="error">Hello</Button.Button>
          <Button.Button status="error" variant="filled">
            Hello
          </Button.Button>
          <Button.Button status="error" variant="text">
            Hello
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <Flex.Box y background={1} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={1}>Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y background={2} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={2}>Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y background={3} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={3}>Hello</Button.Button>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
