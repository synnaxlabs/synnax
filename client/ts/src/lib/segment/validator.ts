import { ContiguityError, UnexpectedError, ValidationError } from '../errors';
import { DataType, TimeStamp, TypedArray } from '../telem';

import TypedSegment from './typed';

export class ScalarTypeValidator {
  validate(array: TypedArray, dataType: DataType): void {
    if (!dataType.checkArray(array)) {
      throw new ValidationError({
        field: 'data',
        message: `Data type mismatch. Expected ${dataType} but got ${array.constructor.name}`,
      });
    }
  }
}

export type ContiguityValidatorProps = {
  allowNoHighWaterMark: boolean;
  allowOverlap: boolean;
  allowGaps: boolean;
};

export class ContiguityValidator {
  highWaterMarks: Map<string, TimeStamp>;
  allowNoHighWaterMark = false;
  allowOverlap = false;
  allowGaps = false;

  constructor(props: ContiguityValidatorProps) {
    this.highWaterMarks = new Map();
    this.allowNoHighWaterMark = props.allowNoHighWaterMark;
    this.allowOverlap = props.allowOverlap;
    this.allowGaps = props.allowGaps;
  }

  validate(segment: TypedSegment): void {
    if (!segment.channel.key) {
      throw new UnexpectedError('Channel key is not set');
    }
    const hwm = this.getHighWaterMark(segment.channel.key);
    if (hwm) {
      this.enforceNoOverlap(hwm, segment);
      this.enforceNoGaps(hwm, segment);
    }
    this.updateHighWaterMark(segment);
  }

  private enforceNoOverlap(hwm: TimeStamp, seg: TypedSegment): void {
    if (!this.allowOverlap && seg.start.before(hwm)) {
      throw new ContiguityError(
        `Segment overlaps with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`
      );
    }
  }

  private enforceNoGaps(hwm: TimeStamp, seg: TypedSegment): void {
    if (!this.allowGaps && !seg.start.equals(hwm)) {
      throw new ContiguityError(
        `Segment is not contiguous with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`
      );
    }
  }

  private getHighWaterMark(key: string): TimeStamp | undefined {
    const hwm = this.highWaterMarks.get(key);
    if (!hwm && !this.allowNoHighWaterMark) {
      throw new UnexpectedError(
        'No high water mark found for channel key ' + key
      );
    }
    return hwm;
  }

  private updateHighWaterMark(seg: TypedSegment): void {
    if (!seg.channel.key) {
      throw new UnexpectedError('Channel key is not set');
    }
    this.highWaterMarks.set(seg.channel.key, seg.end);
  }
}
