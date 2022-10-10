import { ValidationError } from '../errors';
import { Size } from '../telem';

import TypedSegment from './typed';

export default class Splitter {
  threshold: Size;

  constructor(threshold: Size) {
    this.threshold = threshold;
  }

  split(segment: TypedSegment): TypedSegment[] {
    if (segment.size.smallerThan(this.threshold)) return [segment];
    if (!segment.channel.density)
      throw new ValidationError(
        'Cannot split segment because channel density is undefined'
      );
    const splitPoint =
      this.threshold.valueOf() -
      (this.threshold.valueOf() % segment.channel.density.valueOf());
    const truncated = new TypedSegment(segment.channel, {
      ...segment.payload,
      data: segment.payload.data.slice(0, splitPoint),
    });
    const next = new TypedSegment(segment.channel, {
      ...segment.payload,
      data: segment.payload.data.slice(splitPoint),
    });
    return [truncated, ...this.split(next)];
  }
}
