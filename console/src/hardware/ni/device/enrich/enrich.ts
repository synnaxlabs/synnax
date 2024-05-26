import data from "@/hardware/ni/device/enrich/enriched.json";
import { PropertiesDigest, EnrichedProperties } from "@/hardware/ni/device/types";
import { UnknownRecord } from "@synnaxlabs/x";

type PickedEnrichedProperties = Pick<
  EnrichedProperties,
  | "analogInput"
  | "analogOutput"
  | "digitalInputOutput"
  | "digitalInput"
  | "digitalOutput"
>;

export const enrich = (info: PropertiesDigest): EnrichedProperties => {
  const enriched = (data as unknown as UnknownRecord)[info.model] as {
    estimatedPinout: PickedEnrichedProperties;
  };
  return {
    ...info,
    ...enriched.estimatedPinout,
  } as EnrichedProperties;
};
