//go:build windows

package computron

/*
#cgo CFLAGS: -I${SRCDIR}/python_install/include/python3.11 -I${SRCDIR}/python_install/lib/python3.11/site-packages/numpy/core/include
#cgo LDFLAGS: -L${SRCDIR}/python_install/lib/combined -lpython311
*/
import "C"
