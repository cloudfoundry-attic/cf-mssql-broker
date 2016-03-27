// +build !cloudfoundry,!heroku,!exclude-odbc-driver

// ODBC driver has a dependency unixodbc on platform other then windows
// Disable by default the odbc driver for cloudfoundry
// E.g. Installing dependencies for ubuntu, including freetds:

// apt-get -y install unixodbc unixodbc-dev
// apt-get -y install freetds-dev freetds-bin tdsodbc
// echo "[FreeTDS]"										> tds.driver.template
// echo "Description     = Open source FreeTDS driver"	>> tds.driver.template
// echo "Driver          =/usr/local/lib/libtdsodbc.so"	>> tds.driver.template
// odbcinst -i -d -f tds.driver.template

package provisioner

import (
	_ "github.com/alexbrainman/odbc"
)
