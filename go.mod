module github.com/go-quicktest/qt

require (
	github.com/google/go-cmp v0.5.9
	github.com/kr/pretty v0.3.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.6.1 // indirect
)

retract (
	v0.0.3 // First retract attempt, that didn't work.
	v1.3.0 // Published accidentally.
	v1.7.0 // Published accidentally.
	v1.9.0 // Published accidentally.
	v1.14.1 // Published accidentally.
	v1.14.2 // Published accidentally.
	v1.14.3 // Contains retractions only.
)

go 1.18
