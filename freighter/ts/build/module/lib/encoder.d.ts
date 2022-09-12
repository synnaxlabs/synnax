import { Payload } from './transport';
export interface EncoderDecoder {
    contentType: string;
    encode(data: Payload): Uint8Array;
    decode<A>(data: Uint8Array): A;
}
export declare class MsgPackEncoderDecoder implements EncoderDecoder {
    contentType: string;
    encode(payload: unknown): Uint8Array;
    decode<A>(data: Uint8Array): A;
}
export declare class JSONEncoderDecoder implements EncoderDecoder {
    contentType: string;
    encode(payload: unknown): Uint8Array;
    decode<A>(data: Uint8Array): A;
}
export declare const ENCODERS: EncoderDecoder[];
