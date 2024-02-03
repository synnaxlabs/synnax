import data from "@/hardware/configure/ni/enriched.json";
import {
  type EnrichedProperties,
  type PropertiesDigest,
} from "@/hardware/device/new/types";

export const enrich = (info: PropertiesDigest): EnrichedProperties => {
  const enriched = data[info.model];
  return {
    ...info,
    ...enriched.estimatedPinout,
  };
};
