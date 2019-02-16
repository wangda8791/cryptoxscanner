module gitlab.com/crankykernel/cryptoxscanner

require (
	github.com/crankykernel/binanceapi-go v0.0.0-20190215060755-6fd15f619dca
	github.com/gobuffalo/packr v1.22.0
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/mitchellh/go-homedir v1.0.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	golang.org/x/crypto v0.0.0-20190103213133-ff983b9c42bc
)

//replace github.com/crankykernel/binanceapi-go => ../../../binanceapi-go
