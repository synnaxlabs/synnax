package computronx

import (
	"fmt"
	// 	"math"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	lua "github.com/yuin/gopher-lua"
)

type Interpreter struct {
	luaState *lua.LState
	mu       sync.Mutex
}

func New() (*Interpreter, error) {
	L := lua.NewState()
	return &Interpreter{luaState: L}, nil
}

type Calculation struct {
	expression  string
	interpreter *Interpreter
	luaState    *lua.LState
}

func (i *Interpreter) NewCalculation(expr string) (*Calculation, error) {
	if expr == "" {
		return nil, errors.New("empty expression")
	}

	// Precompile the Lua code to validate syntax
	compiled, err := i.luaState.LoadString(fmt.Sprintf("return %s", expr))
	if err != nil {
		return nil, errors.Wrap(err, "invalid Lua syntax")
	}

	// Ensure the expression evaluates to a single value
	if compiled == nil {
		return nil, errors.New("expression did not compile to a valid Lua chunk")
	}

	return &Calculation{expression: expr, interpreter: i}, nil
}

func (c *Calculation) Run(vars map[string]interface{}) (telem.Series, error) {
	L := c.interpreter.luaState
	if L == nil {
		return telem.Series{}, errors.New("Lua state is not initialized")
	}

	// Update variables for current sample
	for k, v := range vars {
		if series, ok := v.(telem.Series); ok && series.Len() > 0 {
			val := telem.ValueAt[float64](series, 0) // Assuming you want the first value
			L.SetGlobal(k, lua.LNumber(val))
		} else {
			return telem.Series{}, errors.Newf("variable %s is not a valid series or is empty", k)
		}
	}

	luaExpr := fmt.Sprintf("return %s", c.expression)
	// fmt.Printf("Evaluating Lua expression: %s\n", luaExpr) // Log the expression
	if err := L.DoString(luaExpr); err != nil {
		return telem.Series{}, errors.Wrapf(err, "failed to evaluate expression: %s", c.expression)
	}

	result := L.Get(-1) // Get result from stack
	L.Pop(1)            // Pop result from stack

	if num, ok := result.(lua.LNumber); ok {
		results := make([]float64, 1)
		results[0] = float64(num) // Assuming you want to store the result
		return telem.NewSeriesV(results...), nil
	} else {
		return telem.Series{}, errors.Newf("expression did not evaluate to a number: %s", c.expression)
	}
}

func (c *Calculation) Close() {
	if c.luaState != nil {
		c.luaState.Close() // Close the Lua state
	}
}
