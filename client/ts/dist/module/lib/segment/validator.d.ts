import { DataType, TimeStamp, TypedArray } from '../telem';
import TypedSegment from './typed';
export declare class ScalarTypeValidator {
    validate(array: TypedArray, dataType: DataType): void;
}
export declare type ContiguityValidatorProps = {
    allowNoHighWaterMark: boolean;
    allowOverlap: boolean;
    allowGaps: boolean;
};
export declare class ContiguityValidator {
    highWaterMarks: Map<string, TimeStamp>;
    allowNoHighWaterMark: boolean;
    allowOverlap: boolean;
    allowGaps: boolean;
    constructor(props: ContiguityValidatorProps);
    validate(segment: TypedSegment): void;
    private enforceNoOverlap;
    private enforceNoGaps;
    private getHighWaterMark;
    private updateHighWaterMark;
}
