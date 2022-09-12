import { decode, encode } from '@msgpack/msgpack';
import { camelKeys, snakeKeys } from './caseconv';
export class MsgPackEncoderDecoder {
    contentType = 'application/msgpack';
    encode(payload) {
        return encode(payload);
    }
    decode(data) {
        return decode(data);
    }
}
export class JSONEncoderDecoder {
    contentType = 'application/json';
    encode(payload) {
        return new TextEncoder().encode(JSON.stringify(snakeKeys(payload)));
    }
    decode(data) {
        return camelKeys(JSON.parse(new TextDecoder().decode(data)));
    }
}
export const ENCODERS = [
    new MsgPackEncoderDecoder(),
    new JSONEncoderDecoder(),
];
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5jb2Rlci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvZW5jb2Rlci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sRUFBRSxNQUFNLGtCQUFrQixDQUFDO0FBRWxELE9BQU8sRUFBRSxTQUFTLEVBQUUsU0FBUyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBU2xELE1BQU0sT0FBTyxxQkFBcUI7SUFDaEMsV0FBVyxHQUFHLHFCQUFxQixDQUFDO0lBRXBDLE1BQU0sQ0FBQyxPQUFnQjtRQUNyQixPQUFPLE1BQU0sQ0FBQyxPQUFPLENBQUMsQ0FBQztJQUN6QixDQUFDO0lBRUQsTUFBTSxDQUFJLElBQWdCO1FBQ3hCLE9BQU8sTUFBTSxDQUFDLElBQUksQ0FBaUIsQ0FBQztJQUN0QyxDQUFDO0NBQ0Y7QUFFRCxNQUFNLE9BQU8sa0JBQWtCO0lBQzdCLFdBQVcsR0FBRyxrQkFBa0IsQ0FBQztJQUVqQyxNQUFNLENBQUMsT0FBZ0I7UUFDckIsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLFNBQVMsQ0FBQyxPQUFPLENBQUMsQ0FBQyxDQUFDLENBQUM7SUFDdEUsQ0FBQztJQUVELE1BQU0sQ0FBSSxJQUFnQjtRQUN4QixPQUFPLFNBQVMsQ0FDZCxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQzNCLENBQUM7SUFDcEIsQ0FBQztDQUNGO0FBRUQsTUFBTSxDQUFDLE1BQU0sUUFBUSxHQUFxQjtJQUN4QyxJQUFJLHFCQUFxQixFQUFFO0lBQzNCLElBQUksa0JBQWtCLEVBQUU7Q0FDekIsQ0FBQyJ9