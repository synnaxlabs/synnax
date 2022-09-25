import {
  camelKeys as _camelKeys,
  snakeKeys as _snakeKeys,
} from 'js-convert-case';

const options = {
  recursive: true,
  recursiveInArray: true,
  keepTypesOnRecursion: [Number, String],
};

export const snakeKeys = (entity: unknown) => _snakeKeys(entity, options);
export const camelKeys = (entity: unknown) => _camelKeys(entity, options);
