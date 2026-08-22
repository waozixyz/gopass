module github.com/waozixyz/pass/gui

go 1.25.0

require (
	github.com/waozixyz/kryon/go/kryon v0.0.0
	github.com/waozixyz/kryon/go/kryui v0.0.0
	github.com/waozixyz/pass v0.0.0
)

require (
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/waozixyz/pass => ..

replace github.com/waozixyz/kryon/go/kryon => ../vendor/kryon/go/kryon

replace github.com/waozixyz/kryon/go/kryui => ./kryui
