"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const ava_1 = __importDefault(require("ava"));
const errors_1 = require("./errors");
class MyCustomError extends errors_1.BaseTypedError {
    constructor(message) {
        super(message, 'MyCustomError');
    }
}
const myCustomErrorEncoder = (error) => {
    return error.message;
};
const myCustomErrorDecoder = (encoded) => {
    return new MyCustomError(encoded);
};
(0, ava_1.default)('isTypedError', (t) => {
    const error = new MyCustomError('test');
    const fError = (0, errors_1.isTypedError)(error);
    t.is(fError, true);
    t.is(error.type, 'MyCustomError');
});
(0, ava_1.default)('encoding an decoding through registry', (t) => {
    (0, errors_1.registerError)({
        type: 'MyCustomError',
        encode: myCustomErrorEncoder,
        decode: myCustomErrorDecoder,
    });
    const error = new MyCustomError('test');
    const encoded = (0, errors_1.encodeError)(error);
    t.is(encoded.type, 'MyCustomError');
    t.is(encoded.data, 'test');
    const decoded = (0, errors_1.assertErrorType)('MyCustomError', (0, errors_1.decodeError)(encoded));
    t.is(decoded.message, 'test');
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXJyb3JzLnNwZWMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL2Vycm9ycy5zcGVjLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7Ozs7O0FBQUEsOENBQXVCO0FBRXZCLHFDQVFrQjtBQUVsQixNQUFNLGFBQWMsU0FBUSx1QkFBYztJQUN4QyxZQUFZLE9BQWU7UUFDekIsS0FBSyxDQUFDLE9BQU8sRUFBRSxlQUFlLENBQUMsQ0FBQztJQUNsQyxDQUFDO0NBQ0Y7QUFFRCxNQUFNLG9CQUFvQixHQUFHLENBQUMsS0FBb0IsRUFBVSxFQUFFO0lBQzVELE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FBQztBQUN2QixDQUFDLENBQUM7QUFFRixNQUFNLG9CQUFvQixHQUFHLENBQUMsT0FBZSxFQUFjLEVBQUU7SUFDM0QsT0FBTyxJQUFJLGFBQWEsQ0FBQyxPQUFPLENBQUMsQ0FBQztBQUNwQyxDQUFDLENBQUM7QUFFRixJQUFBLGFBQUksRUFBQyxjQUFjLEVBQUUsQ0FBQyxDQUFDLEVBQUUsRUFBRTtJQUN6QixNQUFNLEtBQUssR0FBRyxJQUFJLGFBQWEsQ0FBQyxNQUFNLENBQUMsQ0FBQztJQUN4QyxNQUFNLE1BQU0sR0FBRyxJQUFBLHFCQUFZLEVBQUMsS0FBSyxDQUFDLENBQUM7SUFDbkMsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxNQUFNLEVBQUUsSUFBSSxDQUFDLENBQUM7SUFDbkIsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxLQUFLLENBQUMsSUFBSSxFQUFFLGVBQWUsQ0FBQyxDQUFDO0FBQ3BDLENBQUMsQ0FBQyxDQUFDO0FBRUgsSUFBQSxhQUFJLEVBQUMsdUNBQXVDLEVBQUUsQ0FBQyxDQUFDLEVBQUUsRUFBRTtJQUNsRCxJQUFBLHNCQUFhLEVBQUM7UUFDWixJQUFJLEVBQUUsZUFBZTtRQUNyQixNQUFNLEVBQUUsb0JBQW9CO1FBQzVCLE1BQU0sRUFBRSxvQkFBb0I7S0FDN0IsQ0FBQyxDQUFDO0lBQ0gsTUFBTSxLQUFLLEdBQUcsSUFBSSxhQUFhLENBQUMsTUFBTSxDQUFDLENBQUM7SUFDeEMsTUFBTSxPQUFPLEdBQUcsSUFBQSxvQkFBVyxFQUFDLEtBQUssQ0FBQyxDQUFDO0lBQ25DLENBQUMsQ0FBQyxFQUFFLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxlQUFlLENBQUMsQ0FBQztJQUNwQyxDQUFDLENBQUMsRUFBRSxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsTUFBTSxDQUFDLENBQUM7SUFDM0IsTUFBTSxPQUFPLEdBQUcsSUFBQSx3QkFBZSxFQUM3QixlQUFlLEVBQ2YsSUFBQSxvQkFBVyxFQUFDLE9BQU8sQ0FBQyxDQUNyQixDQUFDO0lBQ0YsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxPQUFPLENBQUMsT0FBTyxFQUFFLE1BQU0sQ0FBQyxDQUFDO0FBQ2hDLENBQUMsQ0FBQyxDQUFDIn0=