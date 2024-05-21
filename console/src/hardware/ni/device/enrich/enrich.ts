import data from "@/hardware/ni/device/enrich/enriched.json";
import { PropertiesDigest, EnrichedProperties } from "@/hardware/ni/device/types";

export const enrich = (info: PropertiesDigest): EnrichedProperties => {
  const enriched = data[info.model];
  return {
    ...info,
    ...enriched.estimatedPinout,
  };
};
