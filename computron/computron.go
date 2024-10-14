package computron

/*
#cgo CFLAGS: -I/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/python_install/include/python3.9 -I/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/python_install/lib/python3.9/site-packages/numpy/core/include
#cgo LDFLAGS: -L/Users/emilianobonilla/Desktop/synnaxlabs/synnax/computron/python_install/lib/combined -lpython3.9-combined -ldl
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
static int is_array(PyObject* obj) { return PyArray_Check(obj); }

// Set multiple items in a Python dictionary
void set_dict_items(PyObject* dict, char** keys, PyObject** values, int count) {
    for (int i = 0; i < count; i++) PyDict_SetItemString(dict, keys[i], values[i]);
}

// Wrapper for Py_CompileString
static PyObject* my_PyCompileString(const char *str, const char *filename, int start) {
    return Py_CompileString(str, filename, start);
}
*/
import "C"
import (
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	xembed "github.com/synnaxlabs/x/embed"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	xsync "github.com/synnaxlabs/x/sync"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

const (
	targetPythonVersion             = "3.9.13"
	dirPerm             os.FileMode = 0755
	filePerm            os.FileMode = 0644
)

type Config struct {
	// Instrumentation is used for logging, tracing, and metrics
	alamos.Instrumentation
	// PythonInstallDir is the directory where the embedded Python installation is
	// extracted.
	// [OPTIONAL] [DEFAULT: /tmp/synnax/computron]
	PythonInstallDir string
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for the computron service.
	DefaultConfig = Config{
		PythonInstallDir: filepath.Join(os.TempDir(), "synnax", "computron"),
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("computron")
	validate.NotEmptyString(v, "PythonInstallDir", c.PythonInstallDir)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.PythonInstallDir = override.String(c.PythonInstallDir, other.PythonInstallDir)
	return c
}

type Interpreter struct {
	cfg     Config
	globals *C.PyObject
}

func New(cfgs ...Config) (*Interpreter, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	cfg.L.Info("starting embedded Python service",
		zap.String("install_dir", cfg.PythonInstallDir),
		zap.String("version", targetPythonVersion),
	)
	s := &Interpreter{cfg: cfg}
	if err := s.initPython(); err != nil {
		return nil, err
	}
	err = s.initGlobals()
	s.cfg.L.Info("embedded Python service started successfully")
	return s, err
}

func (s *Interpreter) initPython() error {
	// Check if the directory exists
	installDir := filepath.Join(s.cfg.PythonInstallDir, "python_install")
	if _, err := os.Stat(s.cfg.PythonInstallDir); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check Python installation directory")
	}
	// Read the contents of the VERSION file
	contents, err := os.ReadFile(filepath.Join(installDir, "VERSION"))
	v := string(contents)
	v = strings.ReplaceAll(v, " ", "")
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to read Python version file")
	}
	if strings.Contains(v, targetPythonVersion) {
		// Check if the version is the same as the embedded Python version. If so,
		// everything is already set up and we can return early.
		s.cfg.L.Debug("Python already installed. skipping installation")
	} else {
		s.cfg.L.Debug("extracting embedded Python installation. this may take a few seconds")
		if err := xembed.Extract(
			embeddedPython,
			s.cfg.PythonInstallDir,
			dirPerm,
			filePerm,
		); err != nil {
			return errors.Newf("failed to extract embedded Python files: %v", err)
		}
		s.cfg.L.Debug("embedded Python installation extracted")
	}
	pythonHome := C.CString(installDir)
	defer C.free(unsafe.Pointer(pythonHome))
	wPythonHome := C.Py_DecodeLocale(pythonHome, nil)
	defer C.PyMem_Free(unsafe.Pointer(wPythonHome))

	// Lock the OS thread before initializing Python
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	C.Py_SetPythonHome(wPythonHome)
	C.Py_Initialize()
	// Initialize threading support
	C.PyEval_InitThreads()

	if res := C.init_numpy(); res != 0 {
		return errors.New("failed to initialize NumPy")
	}

	// Release the GIL acquired by PyEval_InitThreads()
	C.PyEval_SaveThread()

	return nil
}

func (s *Interpreter) initGlobals() error {
	// Lock the OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Acquire the GIL
	gstate := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gstate)

	s.globals = C.PyDict_New()
	if s.globals == nil {
		return errors.Newf("failed to create Python globals dictionary")
	}
	// Import NumPy and add it to globals
	numpyName := C.CString("numpy")
	defer C.free(unsafe.Pointer(numpyName))
	numpyModule := C.PyImport_ImportModule(numpyName)
	if numpyModule == nil {
		C.PyErr_Print()
		return fmt.Errorf("failed to import numpy")
	}
	npKey := C.CString("np")
	defer C.free(unsafe.Pointer(npKey))
	C.PyDict_SetItemString(s.globals, npKey, numpyModule)
	C.Py_DecRef(numpyModule)
	return nil
}

type Calculation struct {
	globals  *C.PyObject
	compiled *C.PyObject
}

func (s *Interpreter) NewCalculation(code string) (*Calculation, error) {
	compiled, err := compile(code)
	return &Calculation{compiled: compiled, globals: s.globals}, err
}

var compiledCodeCache xsync.Map[string, *C.PyObject]

func compile(code string) (*C.PyObject, error) {
	// Lock the OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Acquire the GIL
	gstate := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gstate)

	if compiledCode, exists := compiledCodeCache.Load(code); exists {
		return compiledCode, nil
	}
	cCode := C.CString(code)
	defer C.free(unsafe.Pointer(cCode))
	filename := C.CString("<string>")
	defer C.free(unsafe.Pointer(filename))
	compiledCode := C.my_PyCompileString(cCode, filename, C.Py_file_input)
	if compiledCode == nil {
		C.PyErr_Print()
		return nil, errors.Newf("failed to compile code")
	}
	// Increase the reference count to keep it in the cache
	C.Py_IncRef(compiledCode)
	compiledCodeCache.Store(code, compiledCode)
	return compiledCode, nil
}

// Run executes Python code and returns a telem.Series
func (c *Calculation) Run(ctx map[string]interface{}) (telem.Series, error) {
	var s telem.Series

	// Lock the OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Acquire the GIL
	gstate := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gstate)

	localsC := C.PyDict_New()
	if localsC == nil {
		return s, errors.Newf("failed to create Python locals dictionary")
	}
	defer C.Py_DecRef(localsC)

	// Set ctx variables in locals
	if len(ctx) > 0 {
		// Prepare arrays of keys and values
		var (
			count  = len(ctx)
			keys   = make([]*C.char, count)
			values = make([]*C.PyObject, count)
			i      = 0
		)
		for k, v := range ctx {
			ck := C.CString(k)
			keys[i] = ck
			pyObj, ok := v.(*C.PyObject)
			if !ok {
				for j := 0; j <= i; j++ {
					C.free(unsafe.Pointer(keys[j]))
				}
				return s, errors.Newf("value for key %s is not a *C.PyObject", k)
			}
			values[i] = pyObj
			i++
		}
		C.set_dict_items(localsC, &keys[0], &values[0], C.int(count))
		for i := 0; i < count; i++ {
			C.free(unsafe.Pointer(keys[i]))
		}
	}

	// Execute the compiled code object with locals
	ret := C.PyEval_EvalCode(c.compiled, c.globals, localsC)
	if ret == nil {
		C.PyErr_Print()
		return s, errors.New("failed to execute code")
	}
	C.Py_DecRef(ret) // Decrease ref count for the result of execution

	// Retrieve 'result' from locals
	cr := C.CString("result")
	defer C.free(unsafe.Pointer(cr))
	r := C.PyDict_GetItemString(localsC, cr)
	if r == nil {
		// If 'result' not in locals, check in globals (in case code modifies globals)
		r = C.PyDict_GetItemString(c.globals, cr)
		if r == nil {
			return s, errors.New("no 'result' variable in ctx or locals")
		}
	}
	// Increase reference count since we are going to use r
	C.Py_IncRef(r)
	series, err := ToSeries(r)
	C.Py_DecRef(r) // Decrease ref count after use
	return series, err
}

// Map telem.DataType to NumPy data types
var (
	toNP = map[telem.DataType]int{
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
	toDT = map[int]telem.DataType{
		C.NPY_UINT8:   telem.Uint8T,
		C.NPY_UINT16:  telem.Uint16T,
		C.NPY_UINT32:  telem.Uint32T,
		C.NPY_UINT64:  telem.Uint64T,
		C.NPY_INT8:    telem.Int8T,
		C.NPY_INT16:   telem.Int16T,
		C.NPY_INT32:   telem.Int32T,
		C.NPY_INT64:   telem.Int64T,
		C.NPY_FLOAT32: telem.Float32T,
		C.NPY_FLOAT64: telem.Float64T,
	}
)

// NewSeries creates a NumPy array from a telem.Series
func NewSeries(s telem.Series) (*C.PyObject, error) {
	// Lock the OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Acquire the GIL
	gstate := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gstate)

	v, ok := toNP[s.DataType]
	if !ok {
		return nil, fmt.Errorf("unsupported data type: %v", s.DataType)
	}
	if len(s.Data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	length := s.Len()
	dataPtr := unsafe.Pointer(&s.Data[0])
	arr := C.create_arr((*C.char)(dataPtr), C.int(length), C.int(v))
	if arr == nil {
		return nil, fmt.Errorf("failed to create numpy array")
	}
	// Ensure s.Data is not garbage collected prematurely
	runtime.KeepAlive(s.Data)
	return arr, nil
}

// ToSeries converts a NumPy array to a telem.Series
func ToSeries(pyArray *C.PyObject) (telem.Series, error) {
	var s telem.Series
	if pyArray == nil {
		return s, errors.New("pyArray is nil")
	}

	// Lock the OS thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Acquire the GIL
	gState := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gState)

	if C.is_array(pyArray) == 0 {
		return s, errors.Newf("cannot convert non-NumPy object to Series")
	}
	arr := (*C.PyArrayObject)(unsafe.Pointer(pyArray))
	npType := C.PyArray_TYPE(arr)
	dt, found := toDT[int(npType)]
	if !found {
		return s, errors.Newf("unsupported numpy data type: %d", int(npType))
	}
	data := C.PyArray_DATA(arr)
	if data == nil {
		return s, errors.Newf("failed to get data pointer from numpy array")
	}
	dims := C.PyArray_DIMS(arr)
	nDim := int(C.PyArray_NDIM(arr))
	if nDim <= 0 {
		return s, errors.Newf("invalid number of dimensions: %d", nDim)
	}
	length := 1
	dimsSlice := (*[1 << 30]C.npy_intp)(unsafe.Pointer(dims))[:nDim:nDim]
	for i := 0; i < nDim; i++ {
		length *= int(dimsSlice[i])
	}
	var (
		itemSize  = int(C.PyArray_ITEMSIZE(arr))
		totalSize = length * itemSize
		dataBytes = unsafe.Slice((*byte)(data), totalSize)
	)
	runtime.KeepAlive(pyArray)
	return telem.Series{DataType: dt, Data: dataBytes}, nil
}
