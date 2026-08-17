module github.com/waozixyz/gopass/gui

go 1.25.0

require (
	github.com/waozixyz/gopass v0.0.0
	github.com/waozixyz/kryon/go/kryui v0.0.0
)

replace github.com/waozixyz/gopass => ..

replace github.com/waozixyz/kryon/go/kryui => ../vendor/kryon/go/kryui
