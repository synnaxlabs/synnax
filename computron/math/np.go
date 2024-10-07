package math

/*
#cgo CFLAGS: -I/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/math/python_install/include/python3.9 -I/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/math/python_install/lib/python3.9/site-packages/numpy/core/include
#cgo LDFLAGS: -L/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/math/python_install/lib/combined -lpython3.9-combined -ldl
#define PY_SSIZE_T_CLEAN
#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <Python.h>
#include <numpy/arrayobject.h>
#include <stdlib.h>

// Initialize NumPy
static int init_numpy() {
    import_array1(-1);
    return 0;  // Return 0 on success, -1 on failure
}

// Create a NumPy array from data without owning the data
PyObject* create_arr(char* data, int length, int type_) {
    npy_intp dims[1] = {length};
    PyObject *numpy_array = PyArray_SimpleNewFromData(1, dims, type_, data);
    if (numpy_array == NULL) {
        PyErr_Print();
        return NULL;
    }
    // Do not set NPY_ARRAY_OWNDATA since Go owns the data
    return numpy_array;
}

// Check if an object is a NumPy array
static int is_array(PyObject* obj) {
    return PyArray_Check(obj);
}

// Set multiple items in a Python dictionary
void set_dict_items(PyObject* dict, char** keys, PyObject** values, int count) {
    for (int i = 0; i < count; i++) {
        PyDict_SetItemString(dict, keys[i], values[i]);
    }
}

// Wrapper for Py_CompileString
static PyObject* my_PyCompileString(const char *str, const char *filename, int start) {
    return Py_CompileString(str, filename, start);
}
*/
import "C"
import (
	"embed"
	"fmt"
	"github.com/synnaxlabs/x/telem"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/sirupsen/logrus"
)

var (
	initOnce   sync.Once
	initError  error
	globalsC   *C.PyObject
	globalsMtx sync.Mutex
)

var (
	compiledCodeCache = make(map[string]*C.PyObject)
	cacheMutex        sync.Mutex
)

//go:embed all:python_install
var embeddedPython embed.FS

const synnaxPythonInstallDir = "/tmp/synnax"

// Extract embedded Python files
func extractEmbeddedPython() (string, error) {
	if _, err := os.Stat(synnaxPythonInstallDir); err == nil {
		return synnaxPythonInstallDir, nil
	}
	err := os.MkdirAll(synnaxPythonInstallDir, 0755)
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(embeddedPython, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := embeddedPython.ReadFile(path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(synnaxPythonInstallDir, path)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
	return synnaxPythonInstallDir, err
}

// Initialize Python and NumPy
func init() {
	initOnce.Do(func() {
		dir, err := extractEmbeddedPython()
		if err != nil {
			initError = fmt.Errorf("failed to extract embedded Python files: %v", err)
			logrus.Error(initError)
			return
		}
		pythonHome := C.CString(dir + "/python_install")
		defer C.free(unsafe.Pointer(pythonHome))
		wpythonHome := C.Py_DecodeLocale(pythonHome, nil)
		defer C.PyMem_Free(unsafe.Pointer(wpythonHome))
		C.Py_SetPythonHome(wpythonHome)
		C.Py_Initialize()
		res := C.init_numpy()
		if res != 0 {
			initError = fmt.Errorf("failed to initialize NumPy")
			logrus.Error(initError)
			return
		}
		// Initialize globals
		initGlobals()
	})
}

// Initialize the Python globals dictionary and import necessary modules
func initGlobals() {
	globalsMtx.Lock()
	defer globalsMtx.Unlock()
	globalsC = C.PyDict_New()
	if globalsC == nil {
		initError = fmt.Errorf("failed to create Python globals dictionary")
		logrus.Error(initError)
		return
	}
	// Import NumPy and add it to globals
	numpyName := C.CString("numpy")
	defer C.free(unsafe.Pointer(numpyName))
	numpyModule := C.PyImport_ImportModule(numpyName)
	if numpyModule == nil {
		C.PyErr_Print()
		initError = fmt.Errorf("failed to import numpy")
		logrus.Error(initError)
		return
	}
	npKey := C.CString("np")
	defer C.free(unsafe.Pointer(npKey))
	C.PyDict_SetItemString(globalsC, npKey, numpyModule)
	C.Py_DecRef(numpyModule)
}

// Get the compiled code object from the cache or compile it if not present
func getCompiledCode(code string) (*C.PyObject, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if compiledCode, exists := compiledCodeCache[code]; exists {
		return compiledCode, nil
	}

	cCode := C.CString(code)
	defer C.free(unsafe.Pointer(cCode))

	filename := C.CString("<string>")
	defer C.free(unsafe.Pointer(filename))

	compiledCode := C.my_PyCompileString(cCode, filename, C.Py_file_input)
	if compiledCode == nil {
		C.PyErr_Print()
		return nil, fmt.Errorf("failed to compile code")
	}

	// Increase the reference count to keep it in the cache
	C.Py_IncRef(compiledCode)
	compiledCodeCache[code] = compiledCode

	return compiledCode, nil
}

// Execute Python code and return a telem.Series
func Exec(code string, globals map[string]interface{}, locals map[string]interface{}) (telem.Series, error) {
	if initError != nil {
		return telem.Series{}, initError
	}

	compiledCode, err := getCompiledCode(code)
	if err != nil {
		return telem.Series{}, err
	}

	globalsMtx.Lock()
	defer globalsMtx.Unlock()
	// Create a new locals dictionary
	localsC := C.PyDict_New()
	if localsC == nil {
		return telem.Series{}, fmt.Errorf("failed to create Python locals dictionary")
	}
	defer C.Py_DecRef(localsC)

	// Set globals variables
	if len(globals) > 0 {
		// Prepare arrays of keys and values
		count := len(globals)
		keys := make([]*C.char, count)
		values := make([]*C.PyObject, count)
		i := 0
		for k, v := range globals {
			ck := C.CString(k)
			keys[i] = ck
			pyObj, ok := v.(*C.PyObject)
			if !ok {
				// Free allocated C strings
				for j := 0; j <= i; j++ {
					C.free(unsafe.Pointer(keys[j]))
				}
				return telem.Series{}, fmt.Errorf("value for key %s is not a *C.PyObject", k)
			}
			values[i] = pyObj
			i++
		}
		// Call the C function once
		C.set_dict_items(globalsC, &keys[0], &values[0], C.int(count))
		// Free allocated C strings
		for i := 0; i < count; i++ {
			C.free(unsafe.Pointer(keys[i]))
		}
	}

	// Execute the compiled code object
	ret := C.PyEval_EvalCode(compiledCode, globalsC, localsC)
	if ret == nil {
		C.PyErr_Print()
		return telem.Series{}, fmt.Errorf("failed to execute code")
	}
	C.Py_DecRef(ret) // Decrease ref count for the result of execution

	// Retrieve 'result' from locals or globals
	cr := C.CString("result")
	defer C.free(unsafe.Pointer(cr))
	r := C.PyDict_GetItemString(localsC, cr)
	if r == nil {
		r = C.PyDict_GetItemString(globalsC, cr)
		if r == nil {
			return telem.Series{}, fmt.Errorf("no 'result' variable in globals or locals")
		}
	}
	// Increase reference count since we are going to use r
	C.Py_IncRef(r)
	series, err := ToSeries(r)
	C.Py_DecRef(r) // Decrease ref count after use
	if err != nil {
		return telem.Series{}, fmt.Errorf("failed to convert result to series: %v", err)
	}
	return series, nil
}

// Map telem.DataType to NumPy data types
var to_np = map[telem.DataType]int{
	telem.Uint8T:   C.NPY_UINT8,
	telem.Uint16T:  C.NPY_UINT16,
	telem.Uint32T:  C.NPY_UINT32,
	telem.Uint64T:  C.NPY_UINT64,
	telem.Int8T:    C.NPY_INT8,
	telem.Int16T:   C.NPY_INT16,
	telem.Int32T:   C.NPY_INT32,
	telem.Int64T:   C.NPY_INT64,
	telem.Float32T: C.NPY_FLOAT32,
	telem.Float64T: C.NPY_FLOAT64,
}

// Create a NumPy array from a telem.Series without copying data
func New(s telem.Series) (*C.PyObject, error) {
	v, ok := to_np[s.DataType]
	if !ok {
		return nil, fmt.Errorf("unsupported data type: %v", s.DataType)
	}
	if len(s.Data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	length := len(s.Data) / int(s.DataType.Density())
	dataPtr := unsafe.Pointer(&s.Data[0])

	arr := C.create_arr((*C.char)(dataPtr), C.int(length), C.int(v))
	if arr == nil {
		return nil, fmt.Errorf("failed to create numpy array")
	}

	// Ensure s.Data is not garbage collected prematurely
	runtime.KeepAlive(s.Data)

	return arr, nil
}

// Convert a NumPy array to a telem.Series without copying data
func ToSeries(pyArray *C.PyObject) (telem.Series, error) {
	if pyArray == nil {
		return telem.Series{}, fmt.Errorf("pyArray is nil")
	}

	if C.is_array(pyArray) == 0 {
		return telem.Series{}, fmt.Errorf("pyArray is not a numpy array")
	}

	arr := (*C.PyArrayObject)(unsafe.Pointer(pyArray))

	npType := C.PyArray_TYPE(arr)
	var dataType telem.DataType
	found := false
	for t, v := range to_np {
		if v == int(npType) {
			dataType = t
			found = true
			break
		}
	}
	if !found {
		return telem.Series{}, fmt.Errorf("unsupported numpy data type: %d", npType)
	}

	data := C.PyArray_DATA(arr)
	if data == nil {
		return telem.Series{}, fmt.Errorf("failed to get data pointer from numpy array")
	}

	dims := C.PyArray_DIMS(arr)
	ndim := int(C.PyArray_NDIM(arr))
	if ndim <= 0 {
		return telem.Series{}, fmt.Errorf("invalid number of dimensions: %d", ndim)
	}
	length := 1
	dimsSlice := (*[1 << 30]C.npy_intp)(unsafe.Pointer(dims))[:ndim:ndim]
	for i := 0; i < ndim; i++ {
		length *= int(dimsSlice[i])
	}

	itemSize := int(C.PyArray_ITEMSIZE(arr))
	totalSize := length * itemSize

	// Create a Go slice backed by the C array data
	var dataBytes []byte
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&dataBytes))
	sliceHeader.Data = uintptr(unsafe.Pointer(data))
	sliceHeader.Len = totalSize
	sliceHeader.Cap = totalSize

	// Ensure pyArray is not garbage collected prematurely
	runtime.KeepAlive(pyArray)

	return telem.Series{
		DataType: dataType,
		Data:     dataBytes,
	}, nil
}
