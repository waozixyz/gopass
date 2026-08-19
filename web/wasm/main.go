//go:build js && wasm

package main

import (
	"syscall/js"

	password "github.com/waozixyz/pass"
)

func generate(_ js.Value, args []js.Value) any {
	if len(args) != 10 {
		return map[string]any{"error": "invalid browser request"}
	}
	result, err := password.Generate(args[0].String(), args[1].String(), args[2].String(), password.Options{
		Length: args[3].Int(), Counter: uint64(args[4].Int()),
		Lowercase: args[5].Bool(), Uppercase: args[6].Bool(),
		Digits: args[7].Bool(), Symbols: args[8].Bool(), Exclude: args[9].String(),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"password": result}
}

func main() {
	fn := js.FuncOf(generate)
	js.Global().Set("passGenerate", fn)
	js.Global().Get("document").Call("dispatchEvent", js.Global().Get("CustomEvent").New("pass-ready"))
	select {}
}
