module github.com/waozixyz/pass/gui

go 1.25.0

require (
	github.com/waozixyz/pass v0.0.0
	github.com/waozixyz/kryon/go/kryui v0.0.0
)

replace github.com/waozixyz/pass => ..

replace github.com/waozixyz/kryon/go/kryui => ../vendor/kryon/go/kryui
