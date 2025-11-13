package time

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var (
	intervalSymbolName = "interval"
	intervalSymbol     = symbol.Symbol{
		Name: intervalSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: "period", Type: types.I64()}},
		}),
	}
	SymbolResolver = symbol.MapResolver{
		intervalSymbolName: intervalSymbol,
	}
)
