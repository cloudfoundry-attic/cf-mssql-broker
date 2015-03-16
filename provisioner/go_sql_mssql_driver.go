// +build !exclude-mssql-driver

// This driver is a pure go implementation and doesn't have any external dependencies

package provisioner

import (
	_ "github.com/denisenkom/go-mssqldb"
	_ "golang.org/x/crypto/md4" // workaround. Godep will not save this package from go-mssqldb
)
