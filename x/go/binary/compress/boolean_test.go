package compress_test

import (
    "testing"
	"github.com/synnaxlabs/x/binary/compress"
)

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
        result := preCompile(testCase.input)
        if result != testCase.expected {
            t.Errorf("Expected %d, but got %d for input %v", testCase.expected, result, testCase.input)
        }
    }
}

func TestCompression(t *testing.T) {
	testCases := []struct {
		input		[]byte
	} {
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
		compressed, size, err := compress.CompressUp(testCase.input)

		if (err != nil) {
			t.Errorf("Function Errored")
		}

		t.Errorf("Expected %d, but got %d with size %d", testCase.input, compressed, size)

		/*
		for i := range compressed {
			if compressed[i] != testCase.input[i] {
				t.Errorf("Expected %d, but got %d with size %d", testCase.input, compressed, size)
			}
		}
		*/
	}
}

/*
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
		compressed, size, err := CompressUp(testCase.input)
		result, err := DecompressUp(compressed, size)

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
*/