import { camelKeys as _camelKeys, snakeKeys as _snakeKeys } from "js-convert-case";

const options = {
  recursive: true,
  recursiveInArray: true,
  keepTypesOnRecursion: [Number, String, Uint8Array],
};

export const snakeKeys = (entity: unknown): unknown => _snakeKeys(entity, options);
export const camelKeys = (entity: unknown): unknown => _camelKeys(entity, options);
