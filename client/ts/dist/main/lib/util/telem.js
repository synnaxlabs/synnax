"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.randomTypedArray = void 0;
const randomTypedArray = (length, dataType) => {
    // generate random bytes of the correct length
    const bytes = new Uint8Array(length * dataType.density.valueOf());
    for (let i = 0; i < bytes.byteLength; i++) {
        bytes[i] = Math.floor(Math.random() * 256);
    }
    return new dataType.arrayConstructor(bytes.buffer);
};
exports.randomTypedArray = randomTypedArray;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidGVsZW0uanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3V0aWwvdGVsZW0udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7O0FBRU8sTUFBTSxnQkFBZ0IsR0FBRyxDQUM5QixNQUFjLEVBQ2QsUUFBa0IsRUFDTixFQUFFO0lBQ2QsOENBQThDO0lBQzlDLE1BQU0sS0FBSyxHQUFHLElBQUksVUFBVSxDQUFDLE1BQU0sR0FBRyxRQUFRLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxDQUFDLENBQUM7SUFDbEUsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLEVBQUUsQ0FBQyxHQUFHLEtBQUssQ0FBQyxVQUFVLEVBQUUsQ0FBQyxFQUFFLEVBQUU7UUFDekMsS0FBSyxDQUFDLENBQUMsQ0FBQyxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLE1BQU0sRUFBRSxHQUFHLEdBQUcsQ0FBQyxDQUFDO0tBQzVDO0lBQ0QsT0FBTyxJQUFJLFFBQVEsQ0FBQyxnQkFBZ0IsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLENBQUM7QUFDckQsQ0FBQyxDQUFDO0FBVlcsUUFBQSxnQkFBZ0Isb0JBVTNCIn0=