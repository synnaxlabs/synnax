// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/device/Configure.css";

import { type device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Button, Form, Nav, Synnax, Text, Triggers } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useRef, useState } from "react";

import { CSS } from "@/css";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { configurablePropertiesZ, Properties } from "@/hardware/ni/device/types";
import { type Layout } from "@/layout";

export const Configure = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) return;
      return await client.hardware.devices.retrieve<Properties>(layoutKey);
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} onClose={onClose} />;
};

interface ConfigureInternalProps extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device<Properties>;
}

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const generateShortIdentifiers = (name: string): string[] => {
  const words = name.split(" ");
  const identifiers = new Set<string>();

  // Generate initials
  const initials = words.map((word) => word.charAt(0).toLowerCase()).join("");
  identifiers.add(initials);
  identifiers.add(initials.replace(/(.)(.)/g, "$1_$2")); // Insert underscores

  // Generate combinations with numbers
  const regex = /\d+/g;
  const hasNumbers = name.match(regex);

  if (hasNumbers) {
    words.forEach((word, index) => {
      if (regex.test(word)) {
        const abbreviatedWords = words
          .map((w, i) => (i !== index ? w.charAt(0).toLowerCase() : w))
          .join("");
        identifiers.add(abbreviatedWords);
        identifiers.add(abbreviatedWords.replace(/(.)(.)/g, "$1_$2")); // Insert underscores
      }
    });
  }

  // Generate other potential combinations
  const wordAbbreviations = words.map((word) =>
    word.length > 3 ? word.substring(0, 3).toLowerCase() : word.toLowerCase(),
  );
  identifiers.add(wordAbbreviations.join(""));
  identifiers.add(wordAbbreviations.join("_"));

  // Limit length of identifiers
  const filteredIdentifiers = Array.from(identifiers).filter(
    (id) => id.length >= 2 && id.length <= 12,
  );

  return filteredIdentifiers;
};

const ConfigureInternal = ({
  device,
  onClose,
}: ConfigureInternalProps): ReactElement => {
  const methods = Form.use<typeof configurablePropertiesZ>({
    values: {
      name: device.name,
      identifier: "",
    },
    schema: configurablePropertiesZ,
  });

  const client = Synnax.use();

  const [step, setStep] = useState("name");
  const [recommendedIds, setRecommendedIds] = useState<string[]>([]);

  const identifierRef = useRef<HTMLInputElement>(null);

  const { isPending, mutate } = useMutation({
    mutationKey: [client?.key],
    onError: console.error,
    mutationFn: async () => {
      if (client == null) return;
      if (step === "name") {
        if (methods.validate("name")) {
          setStep("identifier");
          setRecommendedIds(
            generateShortIdentifiers(methods.get<string>("name").value),
          );
          setTimeout(() => identifierRef.current?.focus(), 100);
        }
      } else if (step === "identifier") {
        if (!methods.validate("identifier")) return;
        const er = enrich(device.model, device.properties);
        await client.hardware.devices.create({
          ...device,
          configured: true,
          name: methods.get<string>("name").value,
          properties: {
            ...er,
            identifier: methods.get<string>("identifier").value,
          },
        });
        onClose();
      }
    },
  });

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" empty>
      <Form.Form {...methods}>
        <Align.Space
          align="stretch"
          style={{ padding: "5rem" }}
          justify="center"
          grow
          size="large"
        >
          {step === "name" && (
            <>
              <Text.Text level="h4" shade={7}>
                Before you can acquire data from this device, we'll need a few details.
                To start off, enter a name so it's easy to look up later.
              </Text.Text>
              <Form.TextField
                inputProps={{ variant: "natural", level: "h2", autoFocus: true }}
                path="name"
                label="Name"
                autoFocus
              />
            </>
          )}
          {step == "identifier" && (
            <>
              <Text.Text level="h4" shade={7}>
                Next, we'll need a short identifier for{" "}
                {methods.get<string>("name").value}. We'll use this as a prefix for all
                channels associated with this device. We've generated some suggestions
                below.
              </Text.Text>
              <Align.Space direction="y" size="small">
                <Form.TextField
                  inputProps={{ variant: "natural", level: "h2", ref: identifierRef }}
                  path="identifier"
                  label="Identifier"
                  autoFocus
                />
                <Align.Space direction="x">
                  <Button.Icon variant="text" size="small" disabled>
                    <Icon.Bolt />
                  </Button.Icon>
                  {recommendedIds.map((id) => (
                    <Button.Button
                      key={id}
                      variant="suggestion"
                      size="small"
                      onClick={() => {
                        methods.set("identifier", id);
                        identifierRef.current?.focus();
                      }}
                    >
                      {id}
                    </Button.Button>
                  ))}
                </Align.Space>
              </Align.Space>
            </>
          )}
        </Align.Space>
      </Form.Form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            {step === "identifier" ? "To Save" : "To Next"}
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button
            type="submit"
            form="create-workspace"
            loading={isPending}
            disabled={isPending}
            onClick={() => mutate()}
            triggers={[SAVE_TRIGGER]}
          >
            {step === "identifier" ? "Save" : "Next"}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

export const CONFIGURE_LAYOUT_TYPE = "configure_NI";
export type LayoutType = typeof CONFIGURE_LAYOUT_TYPE;

export const createConfigureLayout =
  (device: string, initial: Omit<Partial<Layout.State>, "type">) =>
  (): Layout.State => {
    const { name = "Configure Device", location = "modal", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: initial.key ?? device,
      name,
      icon: "Logo.NI",
      window: {
        navTop: true,
        size: { height: 350, width: 800 },
        resizable: true,
      },
      location,
      ...rest,
    };
  };
