package compress_test

import (
    "testing"
	"github.com/synnaxlabs/x/binary/compress"
)

/*
func TestCountLargestSequence(t *testing.T) {
    testCases := []struct {
        input    []byte
        expected int
    }{
        // Test cases with alternating 0s and 1s
        {[]byte{0, 1, 0, 1, 0, 1}, 1},
        {[]byte{1, 0, 1, 0, 1, 0}, 1},
        
        // Test cases with longest sequence of 0s
        {[]byte{0, 0, 0, 1, 1, 0, 0, 0, 0, 1}, 4},
        {[]byte{0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0}, 4},
        
        // Test cases with longest sequence of 1s
        {[]byte{1, 1, 1, 0, 0, 1, 1, 1, 1, 0}, 4},
        {[]byte{0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1}, 4},
        
        // Edge cases
        {[]byte{0}, 1}, // Single element input
        {[]byte{1}, 1}, // Single element input
        {[]byte{0, 0, 0, 0, 0, 0, 0}, 4}, // All 0s
        {[]byte{1, 1, 1, 1, 1}, 4}, // All 1s
    }

    for _, testCase := range testCases {
        result := compress.preCompile(testCase.input)
        if result != testCase.expected {
            t.Errorf("Expected %d, but got %d for input %v", testCase.expected, result, testCase.input)
        }
    }
}
*/

// First byte includes size 
// Final byte of the array is as far left as it goes
func TestCompression(t *testing.T) {
	testCases := []struct {
		input		[]byte
		answer 		[]byte
	} {
		{[]byte{0, 1, 0, 1, 0, 1}, []byte{1, 252}},
		{[]byte{1, 0, 1, 0, 1, 0}, []byte{1, 126}}, 
		
		// Test cases with longest sequence of 0s
		{[]byte{0, 0, 0, 1, 1, 0, 0, 0, 0, 1}, []byte{4, 50, 65}},
		{[]byte{0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0}, []byte{4, 18, 66, 80}},
		
		// Test cases with longest sequence of 1s
		{[]byte{1, 1, 1, 0, 0, 1, 1, 1, 1, 0}, []byte{4, 3, 36, 16}},
		{[]byte{0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1}, []byte{4, 21, 18, 36}},
		
		// Edge cases
		{[]byte{0}, []byte{1, 128}}, // Single element input
		{[]byte{1}, []byte{1, 64}}, // Single element input
		{[]byte{0, 0, 0, 0, 0, 0, 0}, []byte{4, 112}}, // All 0s
		{[]byte{1, 1, 1, 1, 1}, []byte{4, 5}}, // All 1s
	}

	for _, testCase := range testCases {
		compressed, err := compress.Compress(testCase.input)

		if (err != nil) {
			t.Errorf("Function Errored")
		}


		for i := range compressed {
			if compressed[i] != testCase.answer[i] {
				t.Errorf("Expected %d, but got %d", testCase.answer, compressed)
			}
		}
	}
}


func TestFull(t *testing.T) {
	testCases := []struct {
		input    []byte
	}{
		// Test cases with alternating 0s and 1s
		{[]byte{0, 1, 0, 1, 0, 1}},
		{[]byte{1, 0, 1, 0, 1, 0}},
		
		// Test cases with longest sequence of 0s
		{[]byte{0, 0, 0, 1, 1, 0, 0, 0, 0, 1}},
		{[]byte{0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0}},
		
		// Test cases with longest sequence of 1s
		{[]byte{1, 1, 1, 0, 0, 1, 1, 1, 1, 0}},
		{[]byte{0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1}},
		
		// Edge cases
		{[]byte{0}}, // Single element input
		{[]byte{1}}, // Single element input
		{[]byte{0, 0, 0, 0, 0, 0, 0}}, // All 0s
		{[]byte{1, 1, 1, 1, 1}}, // All 1s
	}

	for _, testCase := range testCases {
		compressed, err := compress.Compress(testCase.input)
		result, err := compress.Decompress(compressed)

		if (err != nil) {
			t.Errorf("Function Errored")
		}

		for i := range result {
			if result[i] != testCase.input[i] {
				t.Errorf("Expected %d, but got %d", testCase.input, result)
			}
		}
	}
}
