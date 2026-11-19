// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { id } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { createRef, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Node } from "@/schematic/node";
import { GROUP } from "@/schematic/node/general/group";
import { StringDisplayForm } from "@/schematic/node/general/stringDisplay/Form";
import { StringDisplay as Primitive } from "@/schematic/node/general/stringDisplay/Primitive";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { stringValue } from "@/vis/stringValue/aether";

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const CONFIG_Z = schematic.stringDisplayNodeConfigZ;

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof CONFIG_Z>({
    values: Node.createConfig({ variant: "string_display" }),
    schema: CONFIG_Z,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

const getBox = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(`.${CSS.B("string-display")}`);
  if (el == null) throw new Error("expected a string display element");
  return el;
};

describe("StringDisplay", () => {
  describe("Primitive", () => {
    it("should render its children inside the box", () => {
      const { container } = render(<Primitive>hello</Primitive>);
      expect(getBox(container).textContent).toBe("hello");
    });

    // The canvas fits the text to the element the ref receives, so the ref must point
    // at the content, not at the box around it.
    it("should pass the content element to the content ref", () => {
      const ref = createRef<HTMLDivElement>();
      render(<Primitive contentRef={ref} />);
      expect(ref.current?.className).toBe(CSS.BE("string-display", "content"));
    });

    it("should keep the configured width", () => {
      const { container } = render(<Primitive inlineSize={100} />);
      expect(getBox(container).style.width).toBe("100px");
    });
  });

  describe("Form", () => {
    const renderForm = (): ReturnType<typeof render> =>
      render(
        <FormWrapper>
          <StringDisplayForm />
        </FormWrapper>,
      );

    it("should open on the telemetry tab", () => {
      const { getByRole } = renderForm();
      expect(getByRole("tab", { name: "Telemetry" }).ariaSelected).toBe("true");
    });

    it("should render the style controls", () => {
      const { getAllByText, getByText } = renderForm();
      fireEvent.click(getByText("Style"));
      expect(getAllByText("Label")).toHaveLength(2);
      expect(getByText("Stroke")).toBeDefined();
      expect(getByText("Text")).toBeDefined();
      expect(getByText("Width")).toBeDefined();
      expect(getAllByText("Size")).toHaveLength(2);
    });

    it("should render the telemetry controls", () => {
      const { getByText } = renderForm();
      expect(getByText("Channel")).toBeDefined();
      expect(getByText("Color")).toBeDefined();
      expect(getByText("Timeout")).toBeDefined();
    });
  });

  // Console's StaticSymbolList filters by group, so a symbol missing from
  // GROUP.symbols is reachable only through search.
  it("should be listed in the general group", () => {
    expect(GROUP.symbols).toContain("string_display");
  });
});

const client = createTestClient();

describe("StringDisplay.Symbol deleted channel", () => {
  it("should say Deleted channel inside its box, with no badge below", async () => {
    const ch = await client.channels.create({
      name: `display_${id.create()}`,
      dataType: DataType.STRING,
      virtual: true,
    });
    await client.channels.delete(ch.key);
    const Wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: stringValue.REGISTRY,
      telemFactories: [],
      renderContext: canvasTest.record(),
    });
    const Display = Node.resolveSpec("string_display").Node;
    const { findByText, container } = render(
      <Wrapper>
        <ReactFlowProvider>
          <Display
            nodeKey="s1"
            selected={false}
            onConfigChange={() => {}}
            config={Node.createConfig({ variant: "string_display", channel: ch.key })}
          />
        </ReactFlowProvider>
      </Wrapper>,
    );
    const text = await findByText("Deleted channel");
    expect(text.closest(`.${CSS.BE("string-display", "content")}`)).not.toBeNull();
    expect(container.querySelector(".lyra-badge")).toBeNull();
  });
});
