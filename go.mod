module gxclient-adapter

replace gxclient-go => github.com/gxchain/gxclient-go v0.0.0-20200415084605-8cea12fe04e1

//replace gxclient-go => /Users/yediwu/GolandProjects/gxclient-go

go 1.13

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.1
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc
	github.com/stretchr/testify v1.5.1
	github.com/tidwall/gjson v1.6.0
	gxclient-go v0.0.0-20200312090254-347b61fbbbdf
)
