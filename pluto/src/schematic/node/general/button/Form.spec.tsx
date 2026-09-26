// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { CSS } from "@/css";
import { Form as Base } from "@/form";
import { Label } from "@/schematic/node/common/label";
import { ButtonForm } from "@/schematic/node/general/button/Form";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Theming } from "@/theming";

const schema = z.object({
  label: Label.configZ,
  color: color.crudeZ.optional(),
  size: z.string(),
});

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const Host = (): ReactElement => {
  const methods = Base.use({
    values: { label: Label.configZ.parse({ label: "Fire" }), size: "medium" },
    schema,
  });
  return (
    <SynnaxWrapper>
      <Base.Form<typeof schema> {...methods}>
        <ButtonForm />
      </Base.Form>
    </SynnaxWrapper>
  );
};

describe("ButtonForm", () => {
  it("should show the primary color on an unset color swatch", () => {
    const c = render(<Host />);
    const swatch = c.container.querySelector<HTMLElement>(`.${CSS.B("color-swatch")}`);
    assert(swatch != null);
    const primary = Theming.themeZ.parse(Theming.SYNNAX_LIGHT).colors.primary.z;
    expect(swatch.style.getPropertyValue(CSS.variable("swatch", "color"))).toEqual(
      color.cssString(primary),
    );
  });
});
