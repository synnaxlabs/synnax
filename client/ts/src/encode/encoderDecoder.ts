export interface EncoderDecoder {
  encode<Type>(input: Type): string;
  decode<Type>(input: string): Type;
}
